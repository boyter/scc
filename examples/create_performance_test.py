# Create folders with files in them to check out the performance of code counters

import os
import errno

code = '''package com.boyter.SpellingCorrector;

import java.util.*;
import java.util.stream.Stream;

public class SpellingCorrector implements ISpellingCorrector {
  private Map<String, Integer> dictionary = null;

  public SpellingCorrector(int lruCount) {
      this.dictionary = Collections.synchronizedMap(new LruCache<>(lruCount));
  }

  @Override
  public void putWord(String word) {
      word = word.toLowerCase();
      if (dictionary.containsKey(word)) {
          dictionary.put(word, (dictionary.get(word) + 1));
      }
      else {
          dictionary.put(word, 1);
      }
  }

  @Override
  public String correct(String word) {
      if (word == null || word.trim().isEmpty()) {
          return word;
      }

      word = word.toLowerCase();

      if (dictionary.containsKey(word)) {
          return word;
      }

      Map<String, Integer> possibleMatches = new HashMap<>();

      List<String> closeEdits = wordEdits(word);
      for (String closeEdit: closeEdits) {
          if (dictionary.containsKey(closeEdit)) {
              possibleMatches.put(closeEdit, this.dictionary.get(closeEdit));
          }
      }

      if (!possibleMatches.isEmpty()) {
          Object[] matches = this.sortByValue(possibleMatches).keySet().toArray();

          String bestMatch = "";
          for(Object o: matches) {
              if (o.toString().length() == word.length()) {
                  bestMatch = o.toString();
              }
          }

          if (!bestMatch.trim().isEmpty()) {
              return bestMatch;
          }

          return matches[matches.length - 1].toString();
      }

      List<String> furtherEdits = new ArrayList<>();
      for(String closeEdit: closeEdits) {
          furtherEdits.addAll(this.wordEdits(closeEdit));
      }

      for (String futherEdit: furtherEdits) {
          if (dictionary.containsKey(futherEdit)) {
              possibleMatches.put(futherEdit, this.dictionary.get(futherEdit));
          }
      }

      if (!possibleMatches.isEmpty()) {
          Object[] matches = this.sortByValue(possibleMatches).keySet().toArray();

          String bestMatch = "";
          for(Object o: matches) {
              if (o.toString().length() == word.length()) {
                  bestMatch = o.toString();
              }
          }

          if (!bestMatch.trim().isEmpty()) {
              return bestMatch;
          }

          return matches[matches.length - 1].toString();
      }

      return word;
  }

  @Override
  public boolean containsWord(String word) {
      if (dictionary.containsKey(word)) {
          return true;
      }

      return false;
  }

  private List<String> wordEdits(String word) {
      List<String> closeWords = new ArrayList<String>();

      for (int i = 1; i < word.length() + 1; i++) {
          for (char character = 'a'; character <= 'z'; character++) {
              StringBuilder sb = new StringBuilder(word);
              sb.insert(i, character);
              closeWords.add(sb.toString());
          }
      }

      for (int i = 1; i < word.length(); i++) {
          for (char character = 'a'; character <= 'z'; character++) {
              StringBuilder sb = new StringBuilder(word);
              sb.setCharAt(i, character);
              closeWords.add(sb.toString());

              sb = new StringBuilder(word);
              sb.deleteCharAt(i);
              closeWords.add(sb.toString());
          }
      }

      return closeWords;
  }

  public static <K, V extends Comparable<? super V>> Map<K, V> sortByValue( Map<K, V> map ) {
      Map<K, V> result = new LinkedHashMap<>();
      Stream<Map.Entry<K, V>> st = map.entrySet().stream();

      st.sorted( Map.Entry.comparingByValue() ).forEachOrdered( e -> result.put(e.getKey(), e.getValue()) );

      return result;
  }

  public class LruCache<A, B> extends LinkedHashMap<A, B> {
      private final int maxEntries;

      public LruCache(final int maxEntries) {
          super(maxEntries + 1, 1.0f, true);
          this.maxEntries = maxEntries;
      }

      @Override
      protected boolean removeEldestEntry(final Map.Entry<A, B> eldest) {
          return super.size() > maxEntries;
      }
  }
}'''

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
    with open(deep_dir + str(x) +'.java', 'w') as myfile:
        myfile.write(code)

# Case 1
# Create a directory thats quite deep and put 100 files in each folder
deep_dir = './'
for x in range(21):
    deep_dir += '1/'
    make_sure_path_exists(deep_dir)
    for x in range(100):
        with open(deep_dir + str(x) +'.java', 'w') as myfile:
            myfile.write(code)

# Case 2
# Create a directory that has a single level and put 10000 files in it
deep_dir = './2/'
make_sure_path_exists(deep_dir)
for x in range(10000):
    with open(deep_dir + str(x) +'.java', 'w') as myfile:
        myfile.write(code)

# Case 3
# Create a directory that has a two levels with 10000 directories in the second with a single file in each
deep_dir = './3/'
make_sure_path_exists(deep_dir)
for x in range(10000):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    with open(tmp_dir + '1.java', 'w') as myfile:
        myfile.write(code)

# Case 4
# Create a directory that with 10 subdirectories and 1000 files in each
deep_dir = './4/'
make_sure_path_exists(deep_dir)
for x in range(10):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(1000):
        with open(tmp_dir + str(x) +'.java', 'w') as myfile:
            myfile.write(code)

# Case 5
# Create a directory that with 20 subdirectories and 500 files in each
deep_dir = './5/'
make_sure_path_exists(deep_dir)
for x in range(20):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(500):
        with open(tmp_dir + str(x) +'.java', 'w') as myfile:
            myfile.write(code)

# Case 6
# Create a directory that with 5 subdirectories and 2000 files in each
deep_dir = './6/'
make_sure_path_exists(deep_dir)
for x in range(5):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(2000):
        with open(tmp_dir + str(x) +'.java', 'w') as myfile:
            myfile.write(code)

# Case 7
# Create a directory that with 100 subdirectories and 100 files in each
deep_dir = './7/'
make_sure_path_exists(deep_dir)
for x in range(100):
    tmp_dir = deep_dir + str(x) + '/'
    make_sure_path_exists(tmp_dir)
    for x in range(100):
        with open(tmp_dir + str(x) +'.java', 'w') as myfile:
            myfile.write(code)
