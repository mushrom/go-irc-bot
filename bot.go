package main

import (
	"fmt"
	"flag"
	"time"
	"strconv"
	"os"
	"bufio"
	"strings"
	"regexp"
	"math/rand"

	"github.com/thoj/go-ircevent"
)

type ircBot struct {
	conn       *irc.Connection;

	links      []string;
	linkfd     *os.File;

	prefix     *string;
	nick       *string;
	channels   *string;
	nickpass   *string;
	server     *string;

	commands   map[string] func(*ircBot, *irc.Event);

	lastmsgs   map[string]string; // map nicks to last messages
	spellcheck Spellchecker;
};

func ping(bot *ircBot, event *irc.Event) {
	curstr := strconv.FormatInt(time.Now().UTC().Unix(), 10);
	bot.conn.Privmsg(event.Arguments[0], event.Nick + ": Pong, " + curstr);
}

func randomLink(bot *ircBot, event *irc.Event) {
	if len(bot.links) == 0 {
		bot.conn.Privmsg(event.Arguments[0], event.Nick + ": fresh out of links, sorry.");

	} else {
		rand.Seed(time.Now().Unix())
		bot.conn.Privmsg(event.Arguments[0], event.Nick + ": " +
						 bot.links[rand.Intn(len(bot.links))]);
	}
}

func printCommands(bot *ircBot, event *irc.Event) {
	temp := "";
	for key := range bot.commands {
		temp += *bot.prefix + key + " ";
	}

	bot.conn.Privmsg(event.Arguments[0],
	event.Nick + ": Current commands: " + temp);
}

func reportBug(bot *ircBot, event *irc.Event) {
	for i := 0; i < 3; i++ {
		fmt.Printf("/!\\ BUG REPORT: <%s> %s\n", event.Nick, event.Message());
	}
	bot.conn.Privmsg(event.Arguments[0], event.Nick + ": Duly noted.");
}

func printPrompt(bot *ircBot) {
	fmt.Print(*bot.nick, "> ")
}

func lineloop(bot *ircBot) {
	scanner := bufio.NewScanner(os.Stdin)
	printPrompt(bot);

	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ");

		switch args[0] {
			case "join":
				if len(args) > 1 { bot.conn.Join(args[1]); }

			case "part":
				if len(args) > 1 { bot.conn.Part(args[1]); }

			case "say":
				if len(args) > 2 {
					temp := strings.Join(args[2:], " ")
					bot.conn.Privmsg(args[1], temp)
				}

			case "quit":
				bot.conn.Quit();

			case "prefix":
				if len(args) > 1 { bot.prefix = &args[1]; }

			case "help": fallthrough
			case "commands":
				fmt.Print(
					"    join [channel]\n",
					"    part [channel]\n",
					"    say [channel] [message ...]\n",
					"    prefix [string]\n",
					"    help | commands\n");
		}

		printPrompt(bot);
	}
}

func readLines(fname string) ([]string, error) {
	var ret []string;
	f, err := os.Open(fname);
	defer f.Close();

	if err != nil {
		return ret, err;
	}

	scanner := bufio.NewScanner(f);
	for scanner.Scan() {
		ret = append(ret, scanner.Text())
	}

	return ret, nil;
}

func loadLinksFile(fname string) (*os.File, []string) {
	var retlinks []string;

	/*
	{
		f, err := os.Open(fname);
		if err == nil {
			scanner := bufio.NewScanner(f);
			for scanner.Scan() {
				retlinks = append(retlinks, scanner.Text())
			}
			f.Close();

		} else {
			fmt.Printf("Couldn't open link database '%s'...\n", fname)
			retlinks = append(retlinks, "https://google.com");
		}
	}
	*/

	retlinks, _ = readLines(fname);
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600);

	if err != nil {
		panic(err);
	}

	return f, retlinks
}

func handleEndOfMOTD(bot *ircBot, event *irc.Event) {
	if *bot.nickpass != "" {
		bot.conn.Privmsg("NickServ", "identify " + *bot.nickpass);
	}

	time.Sleep(3 * time.Second);
	bot.conn.Join(*bot.channels)
	bot.conn.Privmsg(*bot.channels, "testing this!");
}

func printPrivmsgs(bot *ircBot, event *irc.Event) {
	fmt.Printf("\r") // clear input prompt
	fmt.Printf("         | %s <%s> %s\n",
	event.Arguments[0], event.Nick, event.Message())
	printPrompt(bot);
}

func updateLastmsgs(bot *ircBot, event *irc.Event) {
	key := strings.ToLower(event.Nick) + event.Arguments[0];
	bot.lastmsgs[key] = event.Message();
}

func getLastmsg(bot *ircBot, event *irc.Event, nick string) (string, bool) {
	key := nick + event.Arguments[0];
	v, found := bot.lastmsgs[key];

	return v, found
}

func ibipResponder(bot *ircBot, event *irc.Event) {
	if event.Message() == ".bots" {
		bot.conn.Privmsg(event.Arguments[0],
		"Reporting in! [\x032Go\x0f] " +
		"(see " + *bot.prefix + "commands and " + *bot.prefix + "help)")
	}
}

func isPrefix(main, sub []rune) bool {
	if len(sub) >= len(main) {
		return false;
	}

	for i := range sub {
		if sub[i] != main[i] {
			return false;
		}
	}

	return true;
}

func handleCommands(bot *ircBot, event *irc.Event) {
	msg := []rune(event.Message());
	fix := []rune(*bot.prefix);

	if isPrefix(msg, fix) {
		str    := string(msg[len(fix):])
		args   := strings.Split(str, " ");
		fn, ok := bot.commands[args[0]]

		if ok {
			fn(bot, event)
		}
	}

	updateLastmsgs(bot, event);
}

func isLink(word string) bool {
	var linkmatcher = regexp.MustCompile(`http[s]?://[^ ]*`);
	return linkmatcher.MatchString(word);
}

func parseLinks(bot *ircBot, event *irc.Event) {
	var linkmatcher = regexp.MustCompile(`http[s]?://[^ ]*`);
	// ignore dumb *chan links
	var ignore = regexp.MustCompile(`(\.onion|chan\.org|chan\.net)`);

	if linkmatcher.MatchString(event.Message()) {
		matches := linkmatcher.FindAllString(event.Message(), -1);

		fmt.Printf("\r") // clear input prompt
		fmt.Printf("         > have links...? %q\n", matches);

		for _, match := range matches {
			if ignore.MatchString(match) {
				fmt.Printf("[ignored]%s\n", match);
				continue;
			}

			fmt.Fprintf(bot.linkfd, "%s\n", match);
			bot.links = append(bot.links, match);
		}

		bot.linkfd.Sync();
	}
}

type hookpair struct {
	fn   func (*ircBot, *irc.Event);
	hook string;
};

func main() {
	var botto ircBot;

	botto.nick = flag.String("nick", "aircbot", "IRC nickname")
	botto.server = flag.String("server", "irc.rizon.net:6697", "server:port combination")
	botto.nickpass = flag.String("nickpass", "", "Password to identify with NickServ")
	botto.channels = flag.String("channels", "#fugginbot", "Comma-seperated list of channels to join on startup")
	botto.prefix = flag.String("prefix", ";", "Command prefix")
	flag.Parse();

	botto.spellcheck, _ = makeLevDistance("./megadict.txt");
	botto.linkfd, botto.links = loadLinksFile("links.db");
	defer botto.linkfd.Close();

	//var commands map[string] func(*irc.Connection, string, string, string);
	botto.lastmsgs = make(map[string]string);
	botto.commands = map[string] func(*ircBot, *irc.Event) {
		"ping": ping,
		"commands": printCommands,
		"randomlink": randomLink,
		"spellcheck": spellcheckCommand,
		"sp": spellcheckCommand,
		"bug": reportBug,
		"help": helpCommand,
		"8ball": eightballCommand,
	};

	botto.conn = irc.IRC(*botto.nick, *botto.nick);
	botto.conn.UseTLS = true

	err := botto.conn.Connect(*botto.server);
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	callbacks := []hookpair {
		{handleCommands,  "PRIVMSG"},
		{ibipResponder,   "PRIVMSG"},
		{parseLinks,      "PRIVMSG"},
		{printPrivmsgs,   "PRIVMSG"},
		{handleEndOfMOTD, "376"},
	};

	for i := range callbacks {
		pair := callbacks[i]
		printPrompt(&botto);
		fmt.Printf("adding hook for %s @ %#v\n", pair.hook, pair.fn);

		botto.conn.AddCallback(pair.hook, func(event *irc.Event) {
			go pair.fn(&botto, event);
		});
	}

	go botto.conn.Loop()
	lineloop(&botto);
}
