'''
One of script to convert JSON structure from old to new required
to support docstrings and C# escaped strings @"
'''
import json

raw = open('languages.json').read()
languages = json.loads(raw)

converted = {}

for key, value in languages.iteritems():
    newquote = []
    for quote in value['quotes']:
        newquote.append({
            'start': quote[0],
            'end': quote[1],
        })

    value['quotes'] = newquote
    converted[key] = value

print json.dumps(converted)