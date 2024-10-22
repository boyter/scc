import json
import os
from pathlib import Path
import re
import string

# import boto3

from datetime import datetime, timedelta
import time
import math
from shutil import rmtree
from tempfile import gettempdir

# s3 = boto3.client('s3')
bucket_name = 'sloccloccode'

def lambda_handler(event, context):
    if 'path' not in event:
        return {
            "statusCode": 400,
            "statusDescription": "400",
            "isBase64Encoded": False,
            "headers": {
                "Content-Type": "text/html"
            },
            "body": '''You be invalid'''
        }

    filename, url, path = process_path(event['path'])

    if filename == None or url == None or path == None:
        return {
            "statusCode": 400,
            "statusDescription": "400",
            "isBase64Encoded": False,
            "headers": {
                "Content-Type": "text/html"
            },
            "body": '''You be invalid'''
        }

    get_process_file(filename=filename, url=url, path=path)

    with open(Path(gettempdir()) / filename, encoding='utf-8') as f:
        content = f.read()

    j = json.loads(content)
    title = 'Total lines'
    s = format_count(sum([x['Lines'] for x in j]))

    if 'category' in event['queryStringParameters']:
        t = event['queryStringParameters']['category']

        if t == 'code':
            title = 'Code lines'
            s = format_count(sum([x['Code'] for x in j]))
        elif t == 'blanks':
            title = 'Blank lines'
            s = format_count(sum([x['Blank'] for x in j]))
        elif t == 'lines':
            pass # it's the default anyway
        elif t == 'comments':
            title = 'Comments'
            s = format_count(sum([x['Comment'] for x in j]))
        elif t == 'cocomo':
            title = 'COCOMO $'
            wage = '56286'
            if 'avg-wage' in event['queryStringParameters']:
                wage = event['queryStringParameters']['avg-wage']

            if wage.isdigit():
                s = format_count(estimate_cost(sum([x['Code'] for x in j]), int(wage)))
            else:
                s = format_count(estimate_cost(sum([x['Code'] for x in j])))

    text_length = '250'
    if len(s) <= 3:
        text_length = '200'

    return {
        "statusCode": 200,
        "statusDescription": "200 OK",
        "isBase64Encoded": False,
        "headers": {
            "Content-Type": "image/svg+xml;charset=utf-8",
            "Cache-Control": "max-age=86400"
        },
        "body": '''<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="100" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="100" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h69v20H0z"/><path fill="#4c1" d="M69 0h31v20H69z"/><path fill="url(#b)" d="M0 0h100v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"> <text x="355" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="590">''' + title + '''</text><text x="355" y="140" transform="scale(.1)" textLength="590">''' + title +'''</text><text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="''' + text_length + '''">''' + s + '''</text><text x="835" y="140" transform="scale(.1)" textLength="''' + text_length + '''">''' + s + '''</text></g> </svg>'''
    }


def get_process_file(filename, url, path):
    s = boto3.resource('s3')
    o = s.Object(bucket_name, filename)

    try:
        unixtime = s3time_to_unix(o.last_modified)
    except:
        # force an update in this case
        unixtime = time.time() - 186400

    diff = int(time.time() - unixtime)
    if diff < 86400:
        o.download_file(Path(gettempdir()) / filename)
    else:
        clone_and_process(filename=filename, url=url, path=path)


def clone_and_process(filename, url, path):
    import git

    download_scc()

    os.chdir(gettempdir())

    rmtree(Path(gettempdir()) / 'scc-tmp-path')
    git.exec_command('clone', '--depth=1', url, 'scc-tmp-path', cwd='/tmp')

    os.system('./scc -f json -o ' + str(Path(gettempdir()) / filename) + ' scc-tmp-path')

    with open(Path(gettempdir()) / filename, 'rb') as f:
        s3.upload_fileobj(f, bucket_name, filename)

    rmtree(Path(gettempdir()) / 'scc-tmp-path')


def download_scc():
    my_file = Path(gettempdir()) / 'scc'
    if my_file.exists() == False:
        with open(my_file, 'wb') as f:
            s3.download_fileobj(bucket_name, 'scc', f)
    my_file.chmod(0o755)


def s3time_to_unix(last_modified):
    datetime_object = datetime.strptime(str(last_modified).split(' ')[0], '%Y-%m-%d')
    unixtime = time.mktime(datetime_object.timetuple())
    return unixtime


def process_path(path):
    path = re.sub('', '', path, flags=re.MULTILINE )

    s = [clean_string(x) for x in path.lower().split('/') if x != '']

    if len(s) != 3:
        return None, None, None

    # Cheap clean check
    for x in s:
        if x == '':
            return None, None, None

    # URL for cloning
    url = 'https://'

    if s[0] == 'github':
        url += 'github.com/'
    if s[0] == 'bitbucket':
        url += 'bitbucket.org/'
    if s[0] == 'gitlab':
        url += 'gitlab.com/'

    url += s[1] + '/'
    url += s[2] + '.git'

    # File for json
    filename = s[0]
    filename += '.' + s[1]
    filename += '.' + s[2] + '.json'

    # Need path
    path = s[2]

    return (filename, url, path)


def clean_string(s):
    valid = string.ascii_lowercase
    valid += string.digits
    valid += '-'
    valid += '.'
    valid += '_'

    clean = ''

    for c in s:
        if c in valid:
            clean += c

    return clean


def format_count(count):
    ranges = [
        (1e18, 'E'),
        (1e15, 'P'),
        (1e12, 'T'),
        (1e9, 'G'),
        (1e6, 'M'),
        (1e3, 'k'),
    ]

    for x, y in ranges:
        if count >= x:
            t = str(round(count / x, 1))
            if len(t) > 3:
                t = t[:t.find('.')]
            return t + y

    return str(round(count, 1))


# EstimateEffort calculate the effort applied using generic COCOMO2 weighted values
def estimate_effort(slocCount):
    return float(3.2) * math.pow(float(slocCount)/1000, 1.05) * 1


# EstimateCost calculates the cost in dollars applied using generic COCOMO2 weighted values based
# on the average yearly wage
def estimate_cost(slocCount, averageWage=56286):
    return estimate_effort(slocCount) * float(averageWage/12) * float(1.8)


if __name__ == '__main__':

    # last_modified = '2019-06-22 07:13:19+00:00'
    # unixtime = s3time_to_unix(last_modified)

    # diff = int(time.time() - unixtime)
    # print(diff > 86400)
    # if diff < 86400:
    #     print('pull from s3 and return')
    # else:
    #     print('clone him and reprocess')

    x, y, z = process_path('/github/boyter/really-cheap-chatbot/')
    print(x, y, z)
    '''
    https://gitlab.com/esr/loccount.git
    https://bitbucket.org/grumdrig/pq-web.git
    https://github.com/boyter/scc.git
    '''

    print(format_count(100))
    print(format_count(1000))
    print(format_count(2500))
    print(format_count(436465))
    print(format_count(263804))
    print(format_count(86400))

    print(format_count(81.99825581739397))

    print('')

    # with open('/home/bboyter/Projects/scc-lambda/tmp.json', encoding='utf-8') as f:
    #     content = f.read()

    # j = json.loads(content)
    # print('lines: ' + format_count(sum([x['Lines'] for x in j])))
    # print('code: ' + format_count(sum([x['Code'] for x in j])))
    # print('comment: ' + format_count(sum([x['Comment'] for x in j])))
    # print('blank: ' + format_count(sum([x['Blank'] for x in j])))
    # print('complexity: ' + format_count(sum([x['Complexity'] for x in j])))


    print(format_count(estimate_cost(710)))
    # s3 = boto3.resource('s3')
    # o = s3.Object('sloccloccode','github.boyter.really-cheap-chatbot.json')
    # print(o.last_modified)
    # o.download_file('/tmp/github.boyter.really-cheap-chatbot.json')