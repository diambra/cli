#!/usr/bin/env python
import sys
import pkg_resources

PKG = sys.argv[0] if sys.argv[0] else "diambra-engine"

print(pkg_resources.get_distribution(PKG).version)
