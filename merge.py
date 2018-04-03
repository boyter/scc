#!/usr/bin/python
 # -*- coding: utf-8 -*-

import json
import pprint
import codecs
import sys

database1 = None
database2 = None

with open('database_languages.json') as json_data:
    database1 = json.load(json_data)

with open('database_languages2.json') as json_data:
    database2 = json.load(json_data)

output = {}

# Bases pulled from https://github.com/Aaronepower/tokei/blob/fe4b8b3b378692455bb5144ebbeb450a75f92d0d/src/language/language.rs#L45
inherits = {
    'c': {
        'line_comment': [
            '//',
        ],
        'multi_line': [
            ['/*', '*/'],
        ],
        'quotes': [
            ['"', '"'],
        ],
    },
    'func': {
        'multi_line': [
            ['(*', '*)'],
        ],
        'quotes': [
            ['"', '"'],
        ],
    },
    'html': {
        'multi_line': [
            ['<!--', '-->'],
        ],
        'quotes': [
            ['"', '"'],
        ]
    },
    'hash': {
        'line_comment': [
            '#',
        ]
    },
    'haskell': {
        'line_comment': [
            '--',
        ],
        'multi_line': [
            ['{-', '-}'],
        ]
    },
    'pro': {
        'line_comment': [
            '%',
        ],
        'multi_line': [
            ['/*', '*/'],
        ],
        'quotes': [
            ['"', '"'],
        ]
    },
    'blank': {},
}

for key, value in database2['languages'].iteritems():
    extensions = []
    line_comment = []
    multi_line = []
    quotes = []
    name = key
    base = ''
    complexitychecks = []

    if 'name' in value:
        name = value['name']
    if 'extensions' in value:
        extensions = value['extensions']
    if 'line_comment' in value:
        line_comment = value['line_comment']
        complexitychecks = ["for ", "for(", "if ", "if(", "switch ", "while ", "else ", "|| ", "&& ", "!= ", "== "]
    if 'multi_line' in value:
        multi_line = value['multi_line']
        complexitychecks = ["for ", "for(", "if ", "if(", "switch ", "while ", "else ", "|| ", "&& ", "!= ", "== "]
    if 'quotes' in value:
        quotes = value['quotes']
        complexitychecks = ["for ", "for(", "if ", "if(", "switch ", "while ", "else ", "|| ", "&& ", "!= ", "== "]
    if 'base' in value:
        base = value['base']

        if 'line_comment' in inherits[base]:
            line_comment = inherits[base]['line_comment']
            complexitychecks = ["for ", "for(", "if ", "if(", "switch ", "while ", "else ", "|| ", "&& ", "!= ", "== "]
        if 'multi_line' in inherits[base]:
            multi_line = inherits[base]['multi_line']
            complexitychecks = ["for ", "for(", "if ", "if(", "switch ", "while ", "else ", "|| ", "&& ", "!= ", "== "]
        if 'quotes' in inherits[base]:
            quotes = inherits[base]['quotes']
            complexitychecks = ["for ", "for(", "if ", "if(", "switch ", "while ", "else ", "|| ", "&& ", "!= ", "== "]


    output[name] = {
        'extensions': extensions,
        'line_comment': line_comment,
        'multi_line': multi_line,
        'quotes': quotes,
        'complexitychecks': complexitychecks
    }


# Merge in whats missing where we can
for language in database1:
    language_name = language['language']
    language_extensions = language['extensions']

    found = False

    for key, value in output.iteritems():
        if language_name.lower() == key.lower():
            found = True
            for extension in value['extensions']:
                if extension not in language_extensions:
                    language_extensions.append(extension)

    if not found:
        # Check if the extension is somewhere if so ignore
        print 'Not Found', language_name, language_extensions


extension_mapping = {}
for key, value in output.iteritems():
    for extension in value['extensions']:
        if extension in extension_mapping:
            print 'Duplicate Extension for', key, extension
        extension_mapping[extension] = key


with open('languages.json', 'w') as myfile:
    myfile.write(json.dumps(output))


# outputstr = 'var ExtensionToLanguage = map[string]string{'
# for key, value in extension_mapping.iteritems():
#     # If not starts with . add one in
#     outputstr += '''"%s": "%s", ''' % (key, value)
# outputstr += '}'

# print
# print outputstr


# outputstr = 'var LanguageFeatures = map[string]LanguageFeature{' + '\n'
# for key, value in output.iteritems():
#     outputstr += '"' + key + '": LanguageFeature{' + '\n'
#     if key in ['Plain Text', 'Text', 'XML', 'JSON', 'Markdown']:
#         outputstr += 'CountCode: false,'
#         outputstr += 'CheckComplexity: false,'
#     else:
#         outputstr += 'CountCode: true,' + '\n'
#         outputstr += 'CheckComplexity: true,' + '\n'
#         outputstr += '''        ComplexityChecks: [][]byte{
#                 []byte("for "),
#                 []byte("for("),
#                 []byte("if "),
#                 []byte("if("),
#                 []byte("switch "),
#                 []byte("while "),
#                 []byte("else "),
#                 []byte("|| "),
#                 []byte("&& "),
#                 []byte("!= "),
#                 []byte("== "),
#             },
#             ComplexityBytes: []byte{
#                 'f',
#                 'i',
#                 's',
#                 'w',
#                 'e',
#                 '|',
#                 '&',
#                 '!',
#                 '=',
#             },''' + '\n'
#     if 'line_comment' in value and len(value['line_comment']) != 0:
#         outputstr += '''SingleLineComment: [][]byte{''' + '\n'
#         for x in  value['line_comment']:
#             outputstr += '[]byte("%s"),' % x
#         outputstr += '},' + '\n'
#     else:
#         outputstr += 'SingleLineComment: [][]byte{},' + '\n'

#     if 'multi_line' in value and len(value['multi_line']) != 0:
#         outputstr += 'MultiLineComment: []OpenClose{' + '\n'
#         for x in value['multi_line']:
#             outputstr += 'OpenClose{' + '\n'
#             outputstr += 'Open:  []byte("%s"),' % x[0]  + '\n'
#             outputstr += 'Close: []byte("%s"),' % x[1] + '\n'
#             outputstr += '},' + '\n'
#         outputstr += '},' + '\n'
        
#     else:
#         outputstr += 'MultiLineComment: []OpenClose{},' + '\n'

#     if 'quotes' in value and len(value['quotes']) != 0:
#         for x in value['quotes']:
#             outputstr += 'OpenClose{' + '\n'
#             if x[0] == '"':
#                 outputstr += '''Open: []byte("\\""),''' + '\n'
#             else:
#                 outputstr += 'Open: []byte("%s"),' % x[0] + '\n'
#             if x[1] == '"':
#                 outputstr += '''Close: []byte("\\""),''' + '\n'
#             else:
#                 outputstr += 'Close: []byte("%s"),' % x[1] + '\n'

#         outputstr += '},' + '\n'
#     else:
#         outputstr += 'StringChecks: []OpenClose{},},' + '\n'
#     break
# outputstr += '}'


# UTF8Writer = codecs.getwriter('utf8')
# sys.stdout = UTF8Writer(sys.stdout)
# print
# print unicode(outputstr)