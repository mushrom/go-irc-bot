package main

import (
	"github.com/thoj/go-ircevent"
)

var helpStrings = map[string]string {
	"ping":       "Responds with the current unix time in UTC.",
	"commands":   "Display the list of currently loaded commands.",
	"randomlink": "Sends a link seen somewhere in chat before.",
	"spellcheck": "0 arguments will spellcheck your last message. " +
	              "1 argument will spellcheck another user's message if it's "+
				  "an existing nick, or spellcheck the word if not. " +
				  "2+ arguments will spellcheck the given arguments.",
	"sp":         "0 arguments will spellcheck your last message. " +
	              "1 argument will spellcheck another user's message if it's "+
				  "an existing nick, or spellcheck the word if not. " +
				  "2+ arguments will spellcheck the given arguments.",
	"8ball":      "Answers a question using quantum thermodynamic entropy to "+
	              "view into the future.",
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
