#!/usr/bin/env bats
# in-order.bats - the helper's in-order holds a text to its words in their order, and says which word it lacks
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

@test "in-order passes words that stand in the text in their order, however far apart and however padded" {
  in-order '🔀 #3 Fix the thing  👤 ada  🔨 draft 3d  🌿 7-fix' '#3' 'Fix the thing' 'ada' 'draft' '7-fix'
  in-order 'anything'
}

@test "in-order fails a word that is missing, and a word that stands before the one it follows, naming it" {
  run in-order '#3 Fix the thing  ada' '#3' 'bob'
  [ "$status" -eq 1 ]
  [[ "$output" == *"'bob' is not where it belongs in:"* ]]
  run in-order '#3 Fix the thing  ada' 'ada' '#3'
  [ "$status" -eq 1 ]
  [[ "$output" == *"'#3' is not where it belongs in:"* ]]
}

@test "in-order reads a word as text, never as a pattern" {
  run in-order 'a.c' 'a*'
  [ "$status" -eq 1 ]
  in-order 'a*c [x]' 'a*' '[x]'
}

# ----------------------------------------------------------------------------------------------------------------------
