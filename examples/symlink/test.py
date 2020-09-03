# this is a comment.

import os

for e in os.scandir('.'):
	if e.is_file():
		print(e)
