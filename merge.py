import json
import pprint

database1 = None
database2 = None

with open('database_languages.json') as json_data:
    database1 = json.load(json_data)

with open('database_languages2.json') as json_data:
    database2 = json.load(json_data)


output = {}

# Build the master list based on theirs
for key, value in database2['languages'].iteritems():
    if 'extensions' in value:
        name = key
        if 'name' in value:
            name = value['name']
        if 'line_comment' in value:
            # print ">>>>>", value['line_comment']
            pass
        if 'multi_line' in value:
            # print ">>>>>", value['multi_line']
            pass
        if 'quotes' in value:
            print ">>>>>", value['quotes']

        output[name] = {
            'extensions': value['extensions']
        }


# Merge in whats missing where we can
for language in database1:
    language_name = language['language']
    language_extensions = language['extensions']

    found = False

    for key, value in output.iteritems():
        if language_name.lower() == key.lower():
            # print 'Found', language_name 
            # print '1', language_extensions
            # print '2', value['extensions']
    
            found = True
    
            for extension in value['extensions']:
                if extension not in language_extensions:
                    language_extensions.append(extension)

            # print '3', language_extensions

    if not found:
        # Check if the extension is somewhere if so ignore
        print 'Not Found', language_name, language_extensions


extension_mapping = {}
for key, value in output.iteritems():
    for extension in value['extensions']:
        if extension in extension_mapping:
            print 'Duplicate Extension for', key, extension
        extension_mapping[extension] = key


outputstr = 'var ExtensionToLanguage = map[string]string{'
for key, value in extension_mapping.iteritems():
    # If not starts with . add one in
    outputstr += '''"%s": "%s", ''' % (key, value)
outputstr += '}'

print
print outputstr


outputstr = 'var LanguageFeatures = map[string]LanguageFeature{' + '\n'
for key, value in output.iteritems():
    outputstr += '"' + key + '": LanguageFeature{' + '\n'
    if key in ['']:
        outputstr += 'CountCode: false,'
        outputstr += 'CheckComplexity: false,'
    else:
        outputstr += 'CountCode: true,'
        outputstr += 'CheckComplexity: true,'
        outputstr += '''        ComplexityChecks: [][]byte{
                []byte("for "),
                []byte("for("),
                []byte("if "),
                []byte("if("),
                []byte("switch "),
                []byte("while "),
                []byte("else "),
                []byte("|| "),
                []byte("&& "),
                []byte("!= "),
                []byte("== "),
            },
            ComplexityBytes: []byte{
                'f',
                'i',
                's',
                'w',
                'e',
                '|',
                '&',
                '!',
                '=',
            },''' + '\n'
    outputstr += 'SingleLineComment: [][]byte{},' + '\n'
    outputstr += 'MultiLineComment: []OpenClose{},' + '\n'
    outputstr += 'StringChecks: []OpenClose{},},' + '\n'
outputstr += '}'

print
print outputstr