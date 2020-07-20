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

func ping(ircobj *irc.Connection, channel, nick, msg string) {
	curstr := strconv.FormatInt(time.Now().UTC().Unix(), 10);
	ircobj.Privmsg(channel, nick + ": Pong, " + curstr);
}

func randomLink(ircobj *irc.Connection, channel, nick, msg string, links []string) {
	rand.Seed(time.Now().Unix())
	ircobj.Privmsg(channel, nick + ": " + links[rand.Intn(len(links))]);
}

func printPrompt(ircobj *irc.Connection) {
	fmt.Print(ircobj.GetNick(), "> ")
}

func lineloop(ircobj *irc.Connection) {
	scanner := bufio.NewScanner(os.Stdin)
	printPrompt(ircobj);

	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ");

		switch args[0] {
			case "join":
				if len(args) > 1 { ircobj.Join(args[1]); }

			case "part":
				if len(args) > 1 { ircobj.Part(args[1]); }

			case "say":
				if len(args) > 2 {
					temp := strings.Join(args[2:], " ")
					ircobj.Privmsg(args[1], temp)
				}

			case "quit":
				ircobj.Quit();

			case "help": fallthrough
			case "commands":
				fmt.Print(
					"    join [channel]\n",
					"    part [channel]\n",
					"    say [channel] [message ...]\n",
					"    help | commands\n");
		}

		printPrompt(ircobj);
	}
}

func loadLinksFile(fname string) (*os.File, []string) {
	var retlinks []string;

	{
		f, err := os.Open(fname);
		if err == nil {
			scanner := bufio.NewScanner(f);
			for scanner.Scan() {
				fmt.Printf("have link %s\n", scanner.Text())
				retlinks = append(retlinks, scanner.Text())
			}
			f.Close();

		} else {
			fmt.Printf("Couldn't open link database '%s'...\n", fname)
			retlinks = append(retlinks, "https://google.com");
		}
	}

	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600);
	if err != nil {
		panic(err);
	}

	return f, retlinks
}

func main() {
	nick := flag.String("nick", "chaibot", "IRC nickname")
	server := flag.String("server", "irc.rizon.net:6697", "server:port combination")
	nickpass := flag.String("nickpass", "", "Password to identify with NickServ")
	channels := flag.String("channels", "#fugginbot", "Comma-seperated list of channels to join on startup")
	prefix := flag.String("prefix", ";", "Command prefix")
	flag.Parse();

	linkfd, links := loadLinksFile("links.db");
	defer linkfd.Close();

	var commands map[string] func(*irc.Connection, string, string, string);
	commands = map[string] func(*irc.Connection, string, string, string) {
		"ping": ping,
		"commands": func(ircobj *irc.Connection, ch, nick, msg string) {
			temp := "";
			for key := range commands {
				temp += *prefix + key + " ";
			}
			ircobj.Privmsg(ch, nick + ": Current commands: " + temp);
		},
		"randomlink": func(ircobj *irc.Connection, ch, nick, msg string) {
			randomLink(ircobj, ch, nick, msg, links);
		},
	}

	ircobj := irc.IRC(*nick, *nick);
	ircobj.UseTLS = true

	err := ircobj.Connect(*server);
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	ircobj.AddCallback("PRIVMSG", func(event *irc.Event) {
		if event.Message()[0:1] == (*prefix)[0:1] {
			args   := strings.Split(event.Message()[1:], " ");
			fn, ok := commands[args[0]]

			if ok {
				go fn(ircobj, event.Arguments[0],
				      event.Nick, event.Message())
			}
		}
	})

	// IBIP responder
	ircobj.AddCallback("PRIVMSG", func(event *irc.Event) {
		go func(event *irc.Event) {
			if event.Message() == ".bots" {
				ircobj.Privmsg(event.Arguments[0],
				               "Reporting in! [\x032Go\x0f] " +
				               "(see " + *prefix + "commands)")
			}
		}(event);
	})

	// link parser
	ircobj.AddCallback("PRIVMSG", func(event *irc.Event) {
		go func(event *irc.Event) {
			var linkmatcher = regexp.MustCompile(`http[s]?://[^ ]*`);

			if linkmatcher.MatchString(event.Message()) {
				matches := linkmatcher.FindAllString(event.Message(), -1);

				fmt.Printf("\r") // clear input prompt
				fmt.Printf("         > have links...? %q\n", matches);

				for i := range matches {
					fmt.Fprintf(linkfd, "%s\n", matches[i]);
					links = append(links, matches[i]);
				}

				linkfd.Sync();
			}
		}(event);
	})

	// message printer/logger
	ircobj.AddCallback("PRIVMSG", func(event *irc.Event) {
		go func(event *irc.Event) {
			fmt.Printf("\r") // clear input prompt
			fmt.Printf("         %s <%s> %s\n",
			           event.Arguments[0], event.Nick, event.Message())
			printPrompt(ircobj)
		}(event);
	})

	// handle end of MOTD
	ircobj.AddCallback("376", func(event *irc.Event) {
		go func(event *irc.Event) {
			if *nickpass != "" {
				ircobj.Privmsg("NickServ", "identify " + *nickpass);
			}

			time.Sleep(3 * time.Second);
			ircobj.Join(*channels)
			ircobj.Privmsg(*channels, "testing this!");
		}(event);
	})

	go ircobj.Loop()
	lineloop(ircobj);
}
