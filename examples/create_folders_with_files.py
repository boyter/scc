# Create folders with files in them to check out the performance of parallel file walk algorithms
# in really unusual edge cases to determine the best all round algorithm for walking

import os
import errno

def make_sure_path_exists(path):
    try:
        os.makedirs(path)
    except OSError as exception:
        if exception.errno != errno.EEXIST:
            raise

# Case 0
# Create a directory thats quite deep and put a 10000 files at the end
deep_dir = './' + '/'.join(["0" for x in range(21)]) + '/'
make_sure_path_exists(deep_dir)
for x in range(10000):
    with open(deep_dir + str(x) +'.py', 'w') as myfile:
        myfile.write("some content")

# Case 1
# Create a directory thats quite deep and put 100 files in each folder
deep_dir = './'
for x in range(21):
    deep_dir += '1/'
    make_sure_path_exists(deep_dir)
    for x in range(100):
        with open(deep_dir + str(x) +'.py', 'w') as myfile:
            myfile.write("some content")

# Case 2
# Create a directory that has a single level and put 10000 files in it
deep_dir = './2/'
make_sure_path_exists(deep_dir)
for x in range(100):
    with open(deep_dir + str(x) +'.py', 'w') as myfile:
        myfile.write("some content")

# Case 3
# Create a directory that has a two levels with 10000 directories in the second with a single file in each
deep_dir = './3/'
make_sure_path_exists(deep_dir)
for x in range(10000):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    with open(tmp_dir + '1.py', 'w') as myfile:
        myfile.write("some content")

# Case 4
# Create a directory that branches out widely with 10 directories each with 10 directories etc...
deep_dir = './4/'
stack = ['1', '2', '3', '4', '5', '6', '7', '8', '9']
make_sure_path_exists(deep_dir)
for x in range(10000):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    with open(tmp_dir + '1.py', 'w') as myfile:
        myfile.write("some content")