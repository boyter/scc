#!/usr/bin/env python3

"""
Docstrings containing an apostrophe (') are handled incorrectly
The line above is counted as code despite being in the middle of a docstring.
The end of docstring flag seems to be changed to an apostrophe,
which means the next line will not exit the docstring.
"""
# Code containing single quotes will exit the docstring,
# but presuming the quotes are balanced the second
# quote will put us in string scanning mode.
if __name__ == '__main__':
    print('Hello, World!')
# Not counted as a comment

# ^ Not counted as a blank line
# Break out of string scanner with unbalanced single quote: '
    exit(0)
