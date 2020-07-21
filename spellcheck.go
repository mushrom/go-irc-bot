package main

import (
	"strings"
	"strconv"
	"unicode"

	"github.com/thoj/go-ircevent"
)

type Spellchecker interface {
	check(word []rune) (int, []string);
	distance(tester, target []rune) int;
}

type LevDistance struct {
	words [][]rune;
	table [64][64]int;
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

func (lev *LevDistance) check(word []rune) (int, []string) {
	curMin := len(word) + 1;
	var matches []string;

	for _, dictword := range lev.words {
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

func makeLevDistance(fname string) (Spellchecker, error) {
	lines, err := readLines(fname);
	var linerunes [][]rune;

	for _, line := range lines {
		ru := strToRunes(line);
		ru[0] = unicode.ToLower(ru[0]);
		linerunes = append(linerunes, ru);
	}

	ret := LevDistance{linerunes, [64][64]int{}};
	return &ret, err;
}

func doSpellcheck(bot *ircBot, args [][]rune) string {
	msg := "";

	for _, arg := range args {
		dist, matches := bot.spellcheck.check(arg);

		if dist > 0 && !isLink(runesToStr(arg)) {
			msg += "\x1f" + runesToStr(arg) + "\x0f (";

			for i, m := range matches {
				if i > 2 {
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
			msg += runesToStr(arg) + " ";
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
			case ',', '.', '!', '?', '<', '>', '[', ']':
				continue;
			default:
				ret += string(ru);
		}
	}

	return ret;
}

func parseArgStr(str string) []string {
	return stripEmpty(strings.Split(stripPunctuation(str), " "));
}

func spellcheckCommand(bot *ircBot, event *irc.Event) {
	strargs := parseArgStr(event.Message())[1:];
	var args [][]rune;

	// if no arguments are given, spellcheck the last message the user typed
	if len(strargs) == 0 {
		lastmsg, found := getLastmsg(bot, event, event.Nick);

		if !found {
			bot.conn.Privmsg(event.Arguments[0],
			                 event.Nick+": haven't seen anything from you recently mate");
			return;
		}

		strargs = parseArgStr(lastmsg);

	// if one argument is given, assume it's a user
	} else if len(strargs) == 1 {
		lastmsg, found := getLastmsg(bot, event, strargs[0]);
		if !found {
			bot.conn.Privmsg(event.Arguments[0],
			                 event.Nick+": haven't seen anything from them");
			return;
		}

		strargs = parseArgStr(lastmsg);
	}

	for _, arg := range strargs {
		ru := strToRunes(arg);
		ru[0] = unicode.ToLower(ru[0]);
		args = append(args, ru);

		/*
		// trim punctuation from the end of words
		lastchar := arg[len(arg) - 1];

		switch lastchar {
			case ',', ':', '.', ';', '!', '?', '>', '<':
				args = append(args, strToRunes(arg[:len(arg)-1])));
			default:
				args = append(args, strToRunes(strings.ToLower(arg)));
		}
		*/
	}

	bot.conn.Privmsg(event.Arguments[0], "<"+event.Nick+"> " + doSpellcheck(bot, args));
}
