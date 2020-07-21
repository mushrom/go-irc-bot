package main

import (
	"github.com/thoj/go-ircevent"
)

var helpStrings = map[string]string {
	"ping":       "Responds with the current unix time in UTC.",
	"commands":   "Display the list of currently loaded commands.",
	"randomlink": "Sends a link seen somewhere in chat before.",
	"spellcheck": "Given no arguments, will spellcheck your last message, " +
	              "1 argument will interpret the argument as a nick and " +
				  "spellcheck their last message, and 2+ arguments will " +
				  "spellcheck the given arguments.",
	"sp":         "Given no arguments, will spellcheck your last message, " +
	              "1 argument will interpret the argument as a nick and " +
				  "spellcheck their last message, and 2+ arguments will " +
				  "spellcheck the given arguments.",
	"bug":        "Files a bug report.",
	"help":       "Display this help message.",
};

func helpCommand(bot *ircBot, event *irc.Event) {
	args := parseArgStr(event.Message())[1:];

	if len(args) == 0 {
		topics := "";

		for key := range helpStrings {
			topics += key + " ";
		}

		bot.conn.Privmsg(event.Arguments[0], event.Nick + ": Available topics: " + topics);

	} else {
		help, found := helpStrings[args[0]];

		if !found {
			bot.conn.Privmsg(event.Arguments[0],
			                 event.Nick + ": Don't know about that, sorry man");

		} else {
			bot.conn.Privmsg(event.Arguments[0],
			                 event.Nick + ": " + help);
		}
	}
}
