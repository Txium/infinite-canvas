import unittest
import tempfile
from pathlib import Path
from unittest.mock import patch
import server
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

    def test_silent_video_preserves_encoded_video(self):
        args = build_command("input", "out.mp4", "mute", self.meta, 0, 0)
        self.assertEqual(args[args.index("-c:v")+1], "copy")
        self.assertIn("-an", args)
        self.assertNotIn("libx264", args)

    def test_precise_clip_is_explicit(self):
        self.assertIn("copy", build_command("input", "out", "clip", self.meta, 1, 3))
        self.assertIn("libx264", build_command("input", "out", "clip", self.meta, 1, 3, True))

    def test_thumbnails_are_bounded_low_resolution(self):
        args = build_command("input", "out.png", "thumbnails", self.meta, 0, 0)
        self.assertIn("scale=160:90", args[args.index("-vf")+1])
        self.assertIn("tile=8x1", args[args.index("-vf")+1])
        self.assertEqual(args[args.index("-frames:v")+1], "1")

    def test_cleanup_only_expired_inactive_jobs(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            for name in ("job-old", "job-active", "original", "final"):
                (root/name).mkdir()
            with patch.object(server,"TEMP_ROOT",root), patch.object(server,"ACTIVE_DIRS",{"job-active"}):
                server.cleanup_temporary(now=10**12)
            self.assertFalse((root/"job-old").exists())
            for name in ("job-active", "original", "final"):
                self.assertTrue((root/name).exists())


if __name__ == "__main__":
    unittest.main()
