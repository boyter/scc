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
for x in range(10000):
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
# Create a directory that with 10 subdirectories and 1000 files in each
deep_dir = './4/'
make_sure_path_exists(deep_dir)
for x in range(10):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(1000):
        with open(tmp_dir + str(x) +'.py', 'w') as myfile:
            myfile.write("some content")

# Case 5
# Create a directory that with 20 subdirectories and 500 files in each
deep_dir = './5/'
make_sure_path_exists(deep_dir)
for x in range(20):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(500):
        with open(tmp_dir + str(x) +'.py', 'w') as myfile:
            myfile.write("some content")

# Case 6
# Create a directory that with 5 subdirectories and 2000 files in each
deep_dir = './6/'
make_sure_path_exists(deep_dir)
for x in range(5):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(2000):
        with open(tmp_dir + str(x) +'.py', 'w') as myfile:
            myfile.write("some content")

# Case 7
# Create a directory that with 100 subdirectories and 100 files in each
deep_dir = './7/'
make_sure_path_exists(deep_dir)
for x in range(100):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(100):
        with open(tmp_dir + str(x) +'.py', 'w') as myfile:
            myfile.write("some content")
