import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('fsp_deploy', Path(__file__).with_name('fsp-deploy.py'))
deploy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(deploy)
REVISION = 'a' * 40


class ArchiveValidationTests(unittest.TestCase):
    def validate(self, images):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / 'images.tar.gz'
            manifest = []
            with tarfile.open(archive, 'w:gz') as bundle:
                for index, (tags, label) in enumerate(images):
                    filename = f'{index}.json'
                    content = json.dumps({'config': {'Labels': {'org.opencontainers.image.revision': label}}}).encode()
                    info = tarfile.TarInfo(filename)
                    info.size = len(content)
                    bundle.addfile(info, io.BytesIO(content))
                    manifest.append({'RepoTags': tags, 'Config': filename, 'Layers': []})
                content = json.dumps(manifest).encode()
                info = tarfile.TarInfo('manifest.json')
                info.size = len(content)
                bundle.addfile(info, io.BytesIO(content))
            deploy.validate_archive(archive, REVISION)

    def expected(self):
        return [([f'fsp-career-api:{REVISION}'], REVISION), ([f'fsp-career-web:{REVISION}'], REVISION)]

    def test_accepts_both_images_from_the_tested_commit(self):
        self.validate(self.expected())

    def test_cannot_replace_another_services_image(self):
        images = self.expected()
        images[0] = (['ctfd/ctfd:latest'], REVISION)
        with self.assertRaises(ValueError):
            self.validate(images)

    def test_cannot_add_an_alias_to_another_service(self):
        images = self.expected()
        images[0][0].append('redis:8.2.1-alpine')
        with self.assertRaises(ValueError):
            self.validate(images)

    def test_rejects_an_image_built_from_another_commit(self):
        images = self.expected()
        images[1] = (images[1][0], 'b' * 40)
        with self.assertRaises(ValueError):
            self.validate(images)

    def test_rejects_a_missing_frontend(self):
        with self.assertRaises(ValueError):
            self.validate(self.expected()[:1])

    def test_rejects_duplicate_backend_images(self):
        with self.assertRaises(ValueError):
            self.validate([self.expected()[0], self.expected()[0]])


if __name__ == '__main__':
    unittest.main()
