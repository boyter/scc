package com.boyter.SpellingCorrector;

import java.util.*;
import java.util.stream.Stream;

/**
 * A simple spell checker based on a few implementations such as the infamous Peter Noving spell checker and
 * the like. Attempts to be highly performing by never changing the first character since we can assume that the
 * user got that correct.
 */
public class SpellingCorrector implements ISpellingCorrector {

    // word to count map - how may times a word is present - or a weight attached to a word
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

        // If the word exists in our dictionary then return
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
            // Sorted least likely first
            Object[] matches = this.sortByValue(possibleMatches).keySet().toArray();

            // Try to match anything of the same length first
            String bestMatch = "";
            for(Object o: matches) {
                if (o.toString().length() == word.length()) {
                    bestMatch = o.toString();
                }
            }

            if (!bestMatch.trim().isEmpty()) {
                return bestMatch;
            }

            // Just return whatever is the best match
            return matches[matches.length - 1].toString();
        }

        // Ok we did't find anything, so lets run the edits function on the previous results and use those
        // this gives us results which are 2 characters away from whatever was entered
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
            // Sorted least likely first
            Object[] matches = this.sortByValue(possibleMatches).keySet().toArray();

            // Try to match anything of the same length first
            String bestMatch = "";
            for(Object o: matches) {
                if (o.toString().length() == word.length()) {
                    bestMatch = o.toString();
                }
            }

            if (!bestMatch.trim().isEmpty()) {
                return bestMatch;
            }

            // Just return whatever is the best match
            return matches[matches.length - 1].toString();
        }


        // If unable to find something better return the same string
        return word;
    }

    @Override
    public boolean containsWord(String word) {
        if (dictionary.containsKey(word)) {
            return true;
        }

        return false;
    }


    /**
     * Return a list of strings which are words similar to our one which could potentially be misspellings
     * Abuse the fact that a char can be used as an integer
     * Assume that they got the first letter correct for all edits to cut on CPU burn time
     */
    private List<String> wordEdits(String word) {
        List<String> closeWords = new ArrayList<String>();

        for (int i = 1; i < word.length() + 1; i++) {
            for (char character = 'a'; character <= 'z'; character++) {
                // Maybe they forgot to type a letter? Try adding one
                StringBuilder sb = new StringBuilder(word);
                sb.insert(i, character);
                closeWords.add(sb.toString());
            }
        }

        for (int i = 1; i < word.length(); i++) {
            for (char character = 'a'; character <= 'z'; character++) {
                // Maybe they mistyped a single letter? Try replacing them all
                StringBuilder sb = new StringBuilder(word);
                sb.setCharAt(i, character);
                closeWords.add(sb.toString());

                // Maybe they added an extra letter? Try deleting one
                sb = new StringBuilder(word);
                sb.deleteCharAt(i);
                closeWords.add(sb.toString());
            }
        }

        return closeWords;
    }


    /**
     * Sorts a map by value taken from
     * http://stackoverflow.com/questions/109383/sort-a-mapkey-value-by-values-java
     */
    public static <K, V extends Comparable<? super V>> Map<K, V> sortByValue( Map<K, V> map ) {
        Map<K, V> result = new LinkedHashMap<>();
        Stream<Map.Entry<K, V>> st = map.entrySet().stream();

        st.sorted( Map.Entry.comparingByValue() ).forEachOrdered( e -> result.put(e.getKey(), e.getValue()) );

        return result;
    }

    /**
     * A very simple LRU cache implementation that can be used for random data types.
     */
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

}
