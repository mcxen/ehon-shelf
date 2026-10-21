"""PyInstaller build helper: python scripts/build.py"""
import os, sys, PyInstaller.__main__
sep = os.pathsep
args = ["app.py", "--name", "EhonShelf", "--noconfirm", "--clean",
        "--add-data", f"ehon_shelf/web{sep}ehon_shelf/web"]
if sys.platform == "darwin": args += ["--windowed", "--osx-bundle-identifier", "io.github.mcxen.ehonshelf"]
else: args += ["--onefile"]
PyInstaller.__main__.run(args)
