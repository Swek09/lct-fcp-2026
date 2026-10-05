package assessment

import (
	"fmt"
	"math/rand/v2"
)

// The same operations and bounded parameter ranges are used for each family.
// Three of six tasks have answers that depend on freshly generated facts;
// copying the static multiple-choice key alone cannot meet the pass threshold.
func scenario(spec, grade string, slot int, r *rand.Rand, topic string) PrivateQuestion {
	n, m := r.IntN(12)+4, r.IntN(7)+2
	prompt, answer := "", ""
	if slot == 1 {
		switch spec {
		case "backend":
			if grade == "junior" {
				base, cut := r.IntN(50)+10, r.IntN(4)+1
				prompt = fmt.Sprintf("В таблице orders пять строк с total: %d, %d, %d, %d, %d. Что вернёт SELECT COUNT(*) FROM orders WHERE total >= %d? Введите число.", base, base+10, base+20, base+30, base+40, base+cut*10)
				answer = fmt.Sprint(5 - cut)
			} else if grade == "middle" {
				extra := r.IntN(5) + 1
				prompt = fmt.Sprintf("Есть %d пользователей, у каждого ровно %d заказа, и ещё %d пользователя без заказов. Сколько строк вернёт users LEFT JOIN orders ON users.id = orders.user_id? Ключ users.id уникален, фильтров нет.", n, m, extra)
				answer = fmt.Sprint(n*m + extra)
			} else {
				existing, newIDs, repeats := n, m, r.IntN(5)+2
				prompt = fmt.Sprintf("В events уже %d уникальных event_id. Новый пакет содержит каждый из них по %d раза и ещё %d новых event_id, каждый тоже по %d раза. INSERT ... ON CONFLICT(event_id) DO NOTHING применяется ко всему пакету. Сколько строк будет в events после commit?", existing, repeats, newIDs, repeats)
				answer = fmt.Sprint(existing + newIDs)
			}
		case "frontend":
			width, padding, border := n*40, m, 2
			if grade == "junior" {
				prompt = fmt.Sprintf("У блока width: %dpx; padding: %dpx; border: %dpx solid; box-sizing: content-box. Какова общая ширина блока без margin в px?", width, padding, border)
				answer = fmt.Sprint(width + 2*(padding+border))
			} else if grade == "middle" {
				gap := r.IntN(8) + 8
				prompt = fmt.Sprintf("CSS Grid: контейнер шириной %dpx, %d одинаковых колонок, gap: %dpx, без padding и border. Какова ширина колонки в px? Округлите до двух знаков.", width, m, gap)
				answer = fmt.Sprintf("%.2f", float64(width-(m-1)*gap)/float64(m))
			} else {
				min, gap := r.IntN(6)*20+120, 16
				prompt = fmt.Sprintf("В CSS Grid задан repeat(auto-fit, minmax(%dpx, 1fr)), gap: %dpx. Ширина области содержимого %dpx, элементов достаточно. Сколько колонок поместится?", min, gap, width)
				answer = fmt.Sprint(max(1, (width+gap)/(min+gap)))
			}
		case "analytics":
			if grade == "junior" {
				prompt = fmt.Sprintf("У продукта %d платящих пользователей. Каждый совершил ровно %d покупки на 500 ₽. Каков ARPPU за период в рублях? Других покупок нет.", n*100, m)
				answer = fmt.Sprint(m * 500)
			} else if grade == "middle" {
				base := r.IntN(10) + 10
				delta := r.IntN(5) + 1
				prompt = fmt.Sprintf("Метрика выросла с %d до %d. На сколько процентов она выросла относительно исходного значения? Округлите до двух знаков.", base, base+delta)
				answer = fmt.Sprintf("%.2f", float64(delta)*100/float64(base))
			} else {
				size := n * 100
				rateA, rateB := m, m+4
				prompt = fmt.Sprintf("Сегмент A: %d пользователей, конверсия %d%%. Сегмент B: %d пользователей, конверсия %d%%. Какова общая конверсия в процентах? Округлите до двух знаков.", size, rateA, 2*size, rateB)
				answer = fmt.Sprintf("%.2f", float64(rateA+2*rateB)/3)
			}
		}
	} else {
		switch spec {
		case "backend":
			if grade == "junior" {
				prompt = fmt.Sprintf("У кандидата %d приглашений от разных компаний, принято ровно %d. Каждая компания имеет только своё приглашение. Сколько компаний вправе получить email кандидата? Доступ открывается только после принятия.", n+m, n)
				answer = fmt.Sprint(n)
			} else if grade == "middle" {
				prompt = fmt.Sprintf("У компании %d отправленных приглашений и %d принятых. Приглашения адресованы разным кандидатам. Серверный endpoint отдаёт email только по принятым приглашениям этой компании. Сколько email должно быть в ответе?", n, m)
				answer = fmt.Sprint(m)
			} else {
				revoked := r.IntN(m)
				prompt = fmt.Sprintf("В сервисе %d активных сессий пользователя с auth_version=4. В БД версия атомарно увеличена до 5. Пользователь создаёт %d новых сессий версии 5, затем отзывает %d из них. Проверка требует совпадения версии и существования сессии. Сколько сессий успешно пройдут проверку?", n, m, revoked)
				answer = fmt.Sprint(m - revoked)
			}
		case "frontend":
			if grade == "junior" {
				prompt = fmt.Sprintf("В форме %d текстовых input, все id уникальны. У %d из них есть видимый label с совпадающим htmlFor. У остальных нет других средств задания accessible name. Сколько input имеют корректную текстовую подпись?", n+m, n)
				answer = fmt.Sprint(n)
			} else if grade == "middle" {
				hidden := r.IntN(n)
				prompt = fmt.Sprintf("В контейнере %d нативных кнопок с tabindex=0; у %d задан disabled, ещё %d других скрыты через display:none. Группы не пересекаются. У остальных нет ограничений. Сколько кнопок входят в последовательный Tab-порядок?", n+m, m, hidden)
				answer = fmt.Sprint(n - hidden)
			} else {
				removed := r.IntN(min(n, m))
				prompt = fmt.Sprintf("Виртуальный список имеет %d видимых фокусируемых элементов и %d элементов overscan с tabindex=-1. При прокрутке %d видимых элемента удалены, добавлены %d новых с tabindex=0. Сколько элементов теперь входит в Tab-порядок? Все группы не пересекаются.", n, m, removed, removed+2)
				answer = fmt.Sprint(n + 2)
			}
		case "analytics":
			if grade == "junior" {
				prompt = fmt.Sprintf("В выгрузке %d различных event_id. Каждый встречается ровно %d раза. После удаления дублей по event_id сколько строк останется?", n, m)
				answer = fmt.Sprint(n)
			} else if grade == "middle" {
				extra := r.IntN(5) + 1
				prompt = fmt.Sprintf("В users %d пользователей с заказами, у каждого %d заказа, и %d пользователя без заказов. Сколько строк получится после LEFT JOIN orders и WHERE orders.id IS NOT NULL? orders.id уникален и не NULL.", n, m, extra)
				answer = fmt.Sprint(n * m)
			} else {
				missing := r.IntN(m)
				prompt = fmt.Sprintf("Для retention D7 есть %d пользователей зрелой когорты, %d вернулись в день 7. Ещё %d пользователя зарегистрировались вчера; из зрелой когорты у %d отсутствует наблюдение из-за сбоя трекинга. По политике пропуски исключаются из знаменателя, все вернувшиеся имеют наблюдение. Каков знаменатель метрики?", n*10, m, n, missing)
				answer = fmt.Sprint(n*10 - missing)
			}
		}
	}
	return PrivateQuestion{Question: Question{ID: fmt.Sprintf("q%d", slot+1), Prompt: prompt, Topic: topic, Kind: "number", Options: []string{}}, Expected: answer}
}
