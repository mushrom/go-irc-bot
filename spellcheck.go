package main

import (
	"strings"
	"strconv"
	"unicode"

	"github.com/thoj/go-ircevent"
)

type Spellchecker interface {
	check(word string) (int, []string);
	valid(word string) bool;
	distance(tester, target []rune) int;
}

type LevDistance struct {
	words     [][]rune;
	table     [64][64]int;
	// to quickly lookup valid words, most words won't have errors
	stringset map[string]bool;
}

func strToRunes(word string) []rune {
	return append([]rune(word), 0);
}

func runesToStr(word []rune) string {
	return string(word[:len(word) - 1]);
}

func min(a, b int) int {
	if a < b {
		return a;
	} else {
		return b;
	}
}

func max(a, b int) int {
	if a > b {
		return a;
	} else {
		return b;
	}
}

func minimum(things ...int) int {
	ret := things[0];

	for _, v := range things[1:] {
		ret = min(ret, v);
	}

	return ret;
}

func maximum(things ...int) int {
	ret := things[0];

	for _, v := range things[1:] {
		ret = max(ret, v);
	}

	return ret;
}

func (lev *LevDistance) distance(tester, target []rune) int {
	if len(tester) > len(lev.table) || len(target) > len(lev.table) {
		return len(lev.table);
	}

	for i := range tester {
		lev.table[i][0] = i;
	}

	for k := range target {
		lev.table[0][k] = k;
	}

	for i := 1; i < len(tester); i++ {
		for k := 1; k < len(target); k++ {
			neq := 1;
			if tester[i-1] == target[k-1] {
				neq = 0;
			}

			lev.table[i][k] = minimum(lev.table[i-1][k] + 1,
			                          lev.table[i][k-1] + 1,
			                          lev.table[i-1][k-1] + neq);

			// Damerau-Levenshtein distance
			if i >= 2 && k >= 2 && tester[i-1] == target[k-2] &&
			                       tester[i-2] == target[k-1] {
				lev.table[i][k] = min(lev.table[i][k],
			                          lev.table[i-2][k-2] + 1);
			}
		}
	}

	return lev.table[len(tester)-1][len(target)-1];
}

func abs(a int) int {
	if a < 0 {
		return -a;
	} else {
		return a;
	}
}

func (lev *LevDistance) check(sword string) (int, []string) {
	if lev.valid(sword) {
		// return early if there's a valid entry in the hashmap
		return 0, []string{sword};
	}

	word   := strToRunes(sword);
	curMin := len(word) + 1;
	var matches []string;

	for _, dictword := range lev.words {
		if abs(len(word) - len(dictword)) > curMin {
			continue;
		}

		dist := lev.distance(word, dictword);

		if dist < curMin {
			curMin = dist;
			matches = []string{runesToStr(dictword)};

		} else if dist == curMin {
			matches = append(matches, runesToStr(dictword));
		}
	}

	return curMin, matches;
}

func (lev *LevDistance) valid(word string) bool {
	_, found := lev.stringset[word];
	return found;
}

func makeLevDistance(fname string) (Spellchecker, error) {
	lines, err := readLines(fname);
	var linerunes [][]rune;
	set := make(map[string]bool);

	for _, line := range lines {
		ru := strToRunes(line);
		ru[0] = unicode.ToLower(ru[0]);
		linerunes = append(linerunes, ru);
		set[line] = true;
	}

	ret := LevDistance{linerunes, [64][64]int{}, set};
	return &ret, err;
}

func doSpellcheck(bot *ircBot, args []string) string {
	msg := "";

	for _, arg := range args {
		dist, matches := bot.spellcheck.check(arg);

		if dist > 0 && !isLink(arg) {
			msg += "\x1f" + arg + "\x0f (";

			for i, m := range matches {
				if i > 2 && i+1 != len(matches) {
					msg += "\x033...(+" + strconv.Itoa(len(matches) - i) + ")\x0f";
					break;
				}

				msg += "\x033" + m + "\x0f";

				if i + 1 != len(matches) {
					msg += "|"
				}
			}

			msg += ")"
			// uncomment for distance info
			//msg += "[d" + strconv.Itoa(dist) + "]";
			msg += " "

		} else {
			msg += arg + " ";
		}
	}

	return msg;
}

func stripEmpty(xs []string) []string {
	var temp []string;

	for _, s := range xs {
		if len(s) > 0 {
			temp = append(temp, s);
		}
	}

	return temp;
}

func stripPunctuation(str string) string {
	var ret string;

	for _, ru := range str {
		switch ru {
			case ',', '.', '!', '?', '<', '>', '[', ']', '"':
				continue;
			default:
				ret += string(ru);
		}
	}

	return ret;
}

func lowerFirsts(xs []string) []string {
	var ret []string;

	for _, s := range xs  {
		ru := []rune(s);
		ru[0] = unicode.ToLower(ru[0]);
		ret = append(ret, string(ru));
	}

	return ret;
}

func parseArgStr(str string) []string {
	return lowerFirsts(stripEmpty(strings.Split(stripPunctuation(str), " ")));
}

func spellcheckCommand(bot *ircBot, event *irc.Event) {
	strargs := parseArgStr(event.Message())[1:];
	destnick := event.Nick;

	if len(strargs) == 1 {
		destnick = strargs[0];
	}

	if len(strargs) < 2 {
		lastmsg, found := getLastmsg(bot, event, strings.ToLower(destnick));

		if found {
			// target is a user we've seen, so check their last message
			strargs = parseArgStr(lastmsg);

		} else if len(strargs) == 1 {
			// haven't seen the user, so assume it's a word check and
			// correct the destination nick
			destnick = event.Nick;

		} else if len(strargs) == 0 {
			// otherwise this was supposed to check the user's last message
			// but we haven't seen anything from the user yet.

			bot.conn.Privmsg(event.Arguments[0],
			                 event.Nick+": haven't seen anything from you recently.");
			return;
		}

		// otherwise fall through assuming the user meant to spellcheck
		// a specific word.
	}

	bot.conn.Privmsg(event.Arguments[0],
	                 "<"+destnick+"> " + doSpellcheck(bot, strargs));
}
