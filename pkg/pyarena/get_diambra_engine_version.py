#!/usr/bin/env python
import sys
import pkg_resources

PKG = sys.argv[1] if len(sys.argv) > 1 else "diambra-engine"

print(pkg_resources.get_distribution(PKG).version)
