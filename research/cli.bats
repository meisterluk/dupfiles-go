#!/usr/bin/env bats

flunk() {
  { if [ "$#" -eq 0 ]; then cat -
    else echo "$@"
    fi
  } | sed "s:${RBENV_TEST_DIR}:TEST_DIR:g" >&2
  return 1
}


@test "CLI version command" {
	dupfiles version
	expected="$(cat -)"
	if [ "$1" != "$2" ]; then
		{ echo "expected: $1"
		  echo "actual:   $2"
		} | flunk
	fi
}

