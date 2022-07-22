#!/usr/bin/env python
import os
import sys
import diambra.arena

if len(sys.argv) < 2:
    print("Usage: diambra arena check-roms <rom>")
    sys.exit(1)

for arg in sys.argv[1:]:
    diambra.arena.check_game_sha_256(os.path.join(os.getenv('DIAMBRAROMSPATH'), arg))
