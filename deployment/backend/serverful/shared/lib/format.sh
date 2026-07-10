if [[ -t 1 ]]; then
  COLOR_RESET=$'\033[0m'
  COLOR_GREEN=$'\033[32m'
  COLOR_RED=$'\033[31m'
else
  COLOR_RESET=''
  COLOR_GREEN=''
  COLOR_RED=''
fi

step() {
  printf '\n%s%s%s\n' "$COLOR_GREEN" "$1" "$COLOR_RESET"
}

warn() {
  printf '\n%s%s%s\n' "$COLOR_RED" "$1" "$COLOR_RESET" >&2
}

fail() {
  printf '\n%s%s%s\n' "$COLOR_RED" "$1" "$COLOR_RESET" >&2
}
