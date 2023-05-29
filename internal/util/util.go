package util

import (
	"fmt"
	"log"
	"math/rand"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var lastTime time.Time

func init() {
	rand.Seed(time.Now().UnixNano())
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func CalculateDeltaTime() float64 {
	if lastTime.IsZero() {
		lastTime = time.Now()
		return 0
	}
	currentTime := time.Now()
	deltaTime := currentTime.Sub(lastTime).Seconds()
	lastTime = currentTime
	return deltaTime
}

func GenerateRandomName() string {
	noun := Nouns[rand.Intn(len(Nouns))]
	verb1 := AdjectiveVerbs1[rand.Intn(len(AdjectiveVerbs1))]
	verb2 := AdjectiveVerbs2[rand.Intn(len(AdjectiveVerbs2))]
	return fmt.Sprintf("%s-%s-%s", verb1, verb2, noun)
}

func HashAndSalt(pwd string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}
	return string(hash)
}

func IsAlphaNumeric(str string) bool {
	for _, c := range str {
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) {
			return false
		}
	}
	return true
}

var AdjectiveVerbs1 = []string{
	"vacuous",
	"cheerful",
	"harmonious",
	"sassy",
	"unsightly",
	"bearded",
	"irritating",
	"plastic",
	"defective",
	"cool",
	"standing",
	"frumpy",
	"perfect",
	"killing",
	"unnatural",
	"driving",
	"highfalutin",
	"wacky",
	"ghastly",
	"sneaking",
	"puffy",
	"humongous",
	"unbiased",
	"shanking",
	"ubiquitous",
	"magnificent",
	"scratchy",
	"wakeful",
	"minor",
	"broken",
	"elusive",
	"flaccid",
	"retro",
	"charging",
	"deeply",
	"dumb",
	"sordid",
	"godly",
	"ginormous",
	"deranged",
	"thin",
	"living",
	"upset",
	"hard",
	"stout",
	"mythic",
	"crappy",
	"wide",
	"profuse",
	"old",
	"hopeful",
	"somber",
	"ethereal",
	"handheld",
	"oval",
	"picayune",
	"squishy",
	"rubbery",
	"fair",
	"walking",
	"passionate",
	"fleshy",
	"precious",
	"stereotyped",
	"licking",
	"rude",
	"injuring",
	"congested",
	"aboriginal",
	"powerful",
	"apprehensive",
	"parched",
	"weak",
	"imported",
	"icky",
	"precocious",
	"hilarious",
	"sinewy",
	"rigid",
	"gullible",
	"ganking",
	"sweltering",
	"industrious",
	"unarmed",
	"trashy",
	"gleaming",
	"unequaled",
	"oily",
	"shallow",
	"speedhacking",
	"narrow",
	"helpless",
	"clever",
	"ragged",
	"tired",
	"yielding",
	"repeating",
	"jolly",
	"fanatical",
	"courageous",
	"curvy",
	"sandy",
	"obsolete",
	"nauseating",
	"momentous",
	"scarred",
	"womanly",
	"high",
	"huge",
	"flippant",
	"lumpy",
	"boring",
	"clean",
	"wet",
	"whimsical",
	"smashing",
	"fancy",
	"gloomy",
	"casual",
	"chunky",
	"woebegone",
	"obtainable",
	"ahead",
	"undesirable",
	"productive",
	"soggy",
	"solitary",
	"puzzled",
	"nonstop",
	"faint",
	"erratic",
	"foamy",
	"thirsty",
	"screeching",
	"aggro",
	"measly",
	"juicy",
	"clunky",
	"elegant",
	"puny",
	"vulgar",
	"bewildered",
	"sacred",
	"voiceless",
	"equable",
	"verdant",
	"hacking",
	"protective",
	"hardcore",
	"ragequitting",
	"strategic",
	"heavy",
	"loving",
	"armless",
	"soft",
	"red",
	"gastly",
	"excellent",
	"willing",
	"creepy",
	"anachronistic",
	"strafing",
	"disturbed",
	"crazy",
	"mean",
	"fragrant",
	"futuristic",
	"great",
	"painful",
	"infinite",
	"chewing",
	"crafting",
	"hurried",
	"slender",
	"crumpled",
	"panting",
	"jagged",
	"warm",
	"odd",
	"permissible",
	"feeding",
	"tenuous",
	"poking",
	"belligerent",
	"imminent",
	"shaky",
	"hungry",
	"unhealthy",
	"ordinary",
	"disgusting",
	"shouting",
	"vivacious",
	"stalking",
	"fighting",
	"relieved",
	"depressed",
	"filthy",
	"raw",
	"withered",
	"sudden",
	"slippery",
	"orange",
	"eminent",
	"cooperative",
	"magenta",
	"robust",
	"encouraging",
	"expanding",
	"abhorrent",
	"soiled",
	"waggish",
	"divergent",
	"pickpocketing",
	"spicy",
	"chubby",
	"flaky",
	"camping",
	"tracking",
	"interesting",
	"intelligent",
	"premium",
	"honorable",
	"fierce",
	"savage",
	"feral",
	"imperfect",
	"inconclusive",
	"amazing",
	"grouping",
	"wretched",
	"lean",
	"optimal",
	"slow",
	"tiny",
	"parsimonious",
	"yummy",
	"confused",
	"exotic",
	"learning",
	"dying",
	"panoramic",
	"elated",
	"blind",
	"impartial",
	"glowing",
	"evil",
	"monstrous",
	"gutsy",
	"cunning",
	"concerned",
	"indie",
	"common",
	"tame",
	"aberrant",
	"assisting",
	"addictive",
}
var AdjectiveVerbs2 = []string{
	"threatening",
	"handsomely",
	"raiding",
	"charred",
	"splendid",
	"hairy",
	"tested",
	"dashing",
	"turgid",
	"jumbled",
	"interactive",
	"elder",
	"sedate",
	"blue",
	"colossal",
	"friendly",
	"shiny",
	"sloppy",
	"dysfunctional",
	"suffering",
	"endurable",
	"joyous",
	"fragging",
	"flat",
	"dapper",
	"bloody",
	"aloof",
	"fetid",
	"stiff",
	"fragile",
	"lame",
	"goofy",
	"abounding",
	"passive",
	"spawning",
	"tackling",
	"innocent",
	"glamorous",
	"inexpensive",
	"intense",
	"spiritual",
	"flashy",
	"mutant",
	"miscreant",
	"fabulous",
	"targeting",
	"swimming",
	"purple",
	"bald",
	"irate",
	"irradiated",
	"crabby",
	"beautiful",
	"tangy",
	"vast",
	"learned",
	"punching",
	"farming",
	"buff",
	"late",
	"neighborly",
	"cowardly",
	"putrid",
	"good",
	"judicious",
	"geeky",
	"wistful",
	"square",
	"torpid",
	"stuffy",
	"stinky",
	"delicious",
	"healing",
	"powerleveling",
	"brave",
	"skinny",
	"rookie",
	"lazy",
	"short",
	"sweet",
	"lively",
	"boundless",
	"stingy",
	"flawless",
	"overwrought",
	"enchanted",
	"bashful",
	"pretty",
	"groping",
	"drunk",
	"furry",
	"volatile",
	"glistening",
	"heady",
	"vigorous",
	"level",
	"offline",
	"quirky",
	"romantic",
	"clumsy",
	"feeble",
	"haunting",
	"miniscule",
	"backstabbing",
	"master",
	"lewd",
	"calm",
	"gaudy",
	"tactical",
	"wrong",
	"kicking",
	"swampy",
	"jogging",
	"watery",
	"vengeful",
	"cloistered",
	"looting",
	"dripping",
	"automatic",
	"disagreeable",
	"stale",
	"scared",
	"husky",
	"cheating",
	"dispensable",
	"dirty",
	"stupid",
	"rare",
	"lascivious",
	"busy",
	"sad",
	"supreme",
	"chatting",
	"psychotic",
	"whispering",
	"polite",
	"gorgeous",
	"ruthless",
	"throwing",
	"obese",
	"unsuitable",
	"marked",
	"acceptable",
	"repulsive",
	"stimulating",
	"quick",
	"manly",
	"pleasing",
	"aspiring",
	"lubricated",
	"miniature",
	"shining",
	"cloudy",
	"curly",
	"nostalgic",
	"terrible",
	"exploiting",
	"murky",
	"angry",
	"howling",
	"proud",
	"talking",
	"tacky",
	"laughable",
	"green",
	"professional",
	"historical",
	"scary",
	"holy",
	"questing",
	"tightfisted",
	"mad",
	"sucky",
	"sleepy",
	"exclusive",
	"quiet",
	"squeamish",
	"rotund",
	"ritzy",
	"trite",
	"light",
	"silly",
	"natural",
	"cold",
	"faulty",
	"poor",
	"invincible",
	"grinding",
	"skateboarding",
	"questionable",
	"yelling",
	"classy",
	"obedient",
	"exuberant",
	"faithful",
	"gangrenous",
	"labored",
	"ambiguous",
	"itchy",
	"rambunctious",
	"flying",
	"pleasant",
	"meek",
	"little",
	"nerdy",
	"average",
	"arrogant",
	"kind",
	"pathetic",
	"synonymous",
	"bright",
	"indifferent",
	"crusty",
	"determined",
	"modding",
	"sweaty",
	"organic",
	"charming",
	"hostile",
	"kindhearted",
	"witty",
	"elite",
	"jazzy",
	"trusting",
	"succulent",
	"mourning",
	"venomous",
	"nasty",
	"long",
	"placid",
	"capricious",
	"tender",
	"phobic",
	"dubious",
	"known",
	"hitting",
	"abrasive",
	"sultry",
	"maniacal",
	"ceaseless",
	"annoyed",
	"sprinting",
	"typical",
	"deserted",
	"cynical",
	"ancient",
	"acidic",
	"fast",
	"melodic",
	"jumping",
	"shaggy",
	"leveling",
	"glitching",
	"pointless",
	"greasy",
	"chilly",
}
var Nouns = []string{
	"abomination",
	"agent",
	"alien",
	"alucard",
	"android",
	"arachnid",
	"archdemon",
	"archer",
	"archlich",
	"archmage",
	"archon",
	"bouncer",
	"asari",
	"assassin",
	"barbarian",
	"bard",
	"bat",
	"beaver",
	"angel",
	"lonin",
	"beast",
	"lord",
	"begger",
	"berserker",
	"birdperson",
	"blacksmith",
	"boar",
	"archduke",
	"sailor",
	"bugbear",
	"veteran",
	"wyrm",
	"captain",
	"centaur",
	"champion",
	"king",
	"queen",
	"cheater",
	"chicken",
	"outcast",
	"conjurer",
	"cleric",
	"cockatrice",
	"colossus",
	"creature",
	"creeper",
	"critter",
	"crusader",
	"cultist",
	"cyborg",
	"dancer",
	"adult",
	"baboon",
	"demon",
	"diety",
	"adventurer",
	"dinosaur",
	"disciple",
	"djinn",
	"acolyte",
	"dragon",
	"dragonkin",
	"dragoon",
	"drake",
	"hoplite",
	"droid",
	"drow",
	"druid",
	"dwarf",
	"elemental",
	"elf",
	"enchanter",
	"bug",
	"engineer",
	"sphinx",
	"cow",
	"faerie",
	"froglok",
	"gamer",
	"frst",
	"tyrant",
	"ghost",
	"ghoul",
	"changeling",
	"gnoll",
	"gnome",
	"goat",
	"goblin",
	"golem",
	"grandmaster",
	"gremlin",
	"griffon",
	"grunt",
	"guardian",
	"gunner",
	"cyclops",
	"hacker",
	"halfling",
	"hippogriff",
	"hobbit",
	"hobgoblin",
	"hobo",
	"human",
	"hunter",
	"fairy",
	"juggernaut",
	"owl",
	"imp",
	"inquisitor",
	"scout",
	"dustywusty",
	"joker",
	"jester",
	"justicar",
	"hipster",
	"chief",
	"knight",
	"kobold",
	"harpy",
	"kraken",
	"emcl",
	"leader",
	"lemming",
	"leviathan",
	"lich",
	"link",
	"lizardman",
	"camel",
	"mage",
	"magus",
	"marine",
	"viper",
	"marksman",
	"mech",
	"member",
	"merchant",
	"minion",
	"mirelurk",
	"mongrel",
	"monk",
	"monster",
	"ape",
	"banshee",
	"badger",
	"devil",
	"mushroom",
	"mystic",
	"neckbeard",
	"necromancer",
	"ninja",
	"nymph",
	"ogre",
	"ooze",
	"oracle",
	"orc",
	"ork",
	"owlbear",
	"paladin",
	"panda",
	"paragon",
	"steed",
	"pawn",
	"bear",
	"penguin",
	"pet",
	"cube",
	"mastermind",
	"pirate",
	"predator",
	"priest",
	"yeti",
	"construct",
	"cat",
	"pyromancer",
	"dog",
	"ranger",
	"raptor",
	"rat",
	"avatar",
	"reaver",
	"robot",
	"rogue",
	"hound",
	"samurai",
	"celestial",
	"satyr",
	"scion",
	"scorpion",
	"seer",
	"phoenix",
	"shade",
	"shadowknight",
	"shaman",
	"shieldbearer",
	"wretch",
	"shopkeeper",
	"mummy",
	"skeleton",
	"slime",
	"crow",
	"smuggler",
	"snake",
	"sniper",
	"soldier",
	"sanic",
	"sorcerer",
	"specter",
	"spider",
	"spy",
	"fiend",
	"hag",
	"succubus",
	"summoner",
	"chimera",
	"synth",
	"gladiator",
	"commoner",
	"templar",
	"aberration",
	"titan",
	"toad",
	"treant",
	"troglodyte",
	"troll",
	"trooper",
	"undead",
	"vampire",
	"villager",
	"mouse",
	"turtle",
	"warg",
	"gunslinger",
	"warlock",
	"warlord",
	"warrior",
	"werewolf",
	"whale",
	"wisp",
	"witch",
	"witchdoctor",
	"wizard",
	"wolf",
	"wolverine",
	"alchemist",
	"tentacle",
	"wraith",
	"wurm",
	"wyvern",
	"watchman",
	"monstrosity",
	"crocodile",
	"crab",
	"dryad",
	"zerg",
	"zombie",
	"baron",
}
