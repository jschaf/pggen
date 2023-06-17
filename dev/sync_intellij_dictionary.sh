#!/bin/bash
set -euo pipefail

dictFile=".idea/dictionaries/joe.xml"
wordFile="dev/words.dic"

# Check if the files exist
if [[ ! -f "$dictFile" ]]; then
  echo "IntelliJ XML dictionary file not found at $dictFile"
  exit 1
fi

if [[ ! -f "$wordFile" ]]; then
  echo "Word list file not found at $wordFile"
  exit 1
fi

# Extract words from the IntelliJ XML dictionary
words=$(awk -F'[<>]' '/<w>/ {print $3}' "$dictFile")

for word in $words; do
  # Check if the word already exists in the word list
  if ! grep -qw "$word" "$wordFile"; then
    echo "Adding word: $word"
    echo "$word" >>"$wordFile"
  fi
done

# Sort the words in-place
sort -u -o "$wordFile" "$wordFile"

echo "Synced IntelliJ dictionary to dev/words.dic"
