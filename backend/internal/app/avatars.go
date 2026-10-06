package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// Store only small, re-encoded raster images. Original metadata and file names
// never reach the browser, and image bytes are never included in search rows.
func normalizeAvatar(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 512*1024 {
		return nil, huma.Error400BadRequest("Изображение слишком большое")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 2048 || cfg.Height > 2048 {
		return nil, huma.Error400BadRequest("Выберите изображение PNG или JPEG размером не более 2048 × 2048")
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, huma.Error400BadRequest("Не удалось прочитать изображение")
	}
	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
	bounds := src.Bounds()
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			r, g, b, a := src.At(bounds.Min.X+x*cfg.Width/256, bounds.Min.Y+y*cfg.Height/256).RGBA()
			// Composite transparent logos on the brand background.
			dst.SetRGBA(x, y, color.RGBA{uint8((r + uint32(27)*(65535-a)/255) / 257), uint8((g + uint32(28)*(65535-a)/255) / 257), uint8((b + uint32(33)*(65535-a)/255) / 257), 255})
		}
	}
	var out bytes.Buffer
	if err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 88}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type AvatarInfo struct {
	Version int64 `json:"version"`
}

func (a *App) avatarRoutes(api huma.API) {
	register(api, "GET", "/api/avatar", "Моё фото или логотип", func(ctx context.Context, in *struct{}) (*Response[AvatarInfo], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		var version int64
		err = a.DB.QueryRow(ctx, `SELECT avatar_version FROM profiles WHERE user_id=$1 UNION ALL SELECT avatar_version FROM companies WHERE user_id=$1`, u.ID).Scan(&version)
		if err != nil {
			return nil, internal(err)
		}
		return ok(AvatarInfo{version}), nil
	})
	type Upload struct {
		Body struct {
			Image string `json:"image" maxLength:"700000"`
		}
	}
	register(api, "POST", "/api/avatar", "Загрузить фото или логотип", func(ctx context.Context, in *Upload) (*Response[AvatarInfo], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		if err = a.limit(ctx, "avatar:"+u.ID, 30, time.Hour); err != nil {
			return nil, err
		}
		raw, err := base64.StdEncoding.DecodeString(in.Body.Image)
		if err != nil {
			return nil, huma.Error400BadRequest("Не удалось прочитать изображение")
		}
		raw, err = normalizeAvatar(raw)
		if err != nil {
			return nil, err
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		_, err = tx.Exec(ctx, `INSERT INTO avatars(user_id,image) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET image=EXCLUDED.image`, u.ID, raw)
		if err != nil {
			return nil, internal(err)
		}
		table := "profiles"
		if u.Role == "employer" {
			table = "companies"
		}
		version := time.Now().UnixMilli()
		_, err = tx.Exec(ctx, "UPDATE "+table+" SET avatar_version=$2 WHERE user_id=$1", u.ID, version)
		if err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return ok(AvatarInfo{version}), nil
	})
	register(api, "DELETE", "/api/avatar", "Удалить моё фото или логотип", func(ctx context.Context, in *struct{}) (*Response[Message], error) {
		u, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		tx, err := a.DB.Begin(ctx)
		if err != nil {
			return nil, internal(err)
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx, "DELETE FROM avatars WHERE user_id=$1", u.ID); err != nil {
			return nil, internal(err)
		}
		table := "profiles"
		if u.Role == "employer" {
			table = "companies"
		}
		if _, err = tx.Exec(ctx, "UPDATE "+table+" SET avatar_version=0 WHERE user_id=$1", u.ID); err != nil {
			return nil, internal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, internal(err)
		}
		return msg("Изображение удалено"), nil
	})
	type ImageOutput struct {
		ContentType string `header:"Content-Type"`
		Body        []byte
	}
	type ImageInput struct {
		ID string `path:"id"`
	}
	register(api, "GET", "/api/avatars/{id}", "Фото доступного профиля или логотип компании", func(ctx context.Context, in *ImageInput) (*ImageOutput, error) {
		viewer, err := a.require(ctx, "")
		if err != nil {
			return nil, err
		}
		var raw []byte
		// Candidate photographs follow exactly the profile's publication and consent
		// rules, including revocation. Company logos are visible to verified users.
		err = a.DB.QueryRow(ctx, `SELECT av.image FROM avatars av JOIN users u ON u.id=av.user_id
   WHERE av.user_id=$1 AND (av.user_id=$2 OR (u.role='employer' AND u.verified) OR ($3='employer' AND EXISTS(SELECT 1 FROM profiles p WHERE p.user_id=av.user_id AND (`+candidateAccess+`))))`, in.ID, viewer.ID, viewer.Role).Scan(&raw)
		if isMissing(err) {
			return nil, huma.Error404NotFound("Изображение не найдено")
		}
		if err != nil {
			return nil, internal(err)
		}
		return &ImageOutput{ContentType: "image/jpeg", Body: raw}, nil
	})
	api.OpenAPI().Paths["/api/avatars/{id}"].Get.Responses["200"].Content = map[string]*huma.MediaType{"image/jpeg": {Schema: &huma.Schema{Type: "string", Format: "binary"}}}
}
