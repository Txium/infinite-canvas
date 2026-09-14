import unittest
from server import build_command


class MediaCommands(unittest.TestCase):
    meta = {"duration": 15, "audioCodec": "aac", "subtitleTracks": 1}

    def test_full_resolution_frame(self):
        args = build_command("input", "output.png", "frame", self.meta, 4.2, 0)
        self.assertIn("4.2", args)
        self.assertNotIn("scale", " ".join(args))
        self.assertIn("png", args)

    def test_copy_audio_and_subtitles(self):
        for action in ("audio", "subtitles"):
            self.assertIn("copy", build_command("input", "out", action, self.meta, 0, 0))

    def test_bad_timestamps(self):
        for start, end in [(3, 2), (-1, 4), (0, 20), (float("nan"), 4)]:
            with self.assertRaises(ValueError):
                build_command("input", "out", "clip", self.meta, start, end)

    def test_precise_clip_is_explicit(self):
        self.assertIn("copy", build_command("input", "out", "clip", self.meta, 1, 3))
        self.assertIn("libx264", build_command("input", "out", "clip", self.meta, 1, 3, True))


if __name__ == "__main__":
    unittest.main()
