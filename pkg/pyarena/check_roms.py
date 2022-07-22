#!/usr/bin/env python
import diambraArena, os, sys

if len(sys.argv) < 2:
    print("Usage: diambra arena check-roms <rom>")
    sys.exit(1)

for arg in sys.argv[1:]:
    diambraArena.checkGameSha256(os.path.join(os.getenv('DIAMBRAROMSPATH'), arg))