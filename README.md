# Sculk-CLI

## Install

You need [Go](https://go.dev/dl/) (the version listed in `go.mod`) and Git.

**Linux / macOS / Codespaces**
```shell
git clone https://github.com/IronCrest-sudo/sculk-cli.git
cd sculk-cli
./install.sh
```
This builds sculk and puts it on your PATH, so you can type `sculk` from anywhere (not `./sculk`). Nothing is left behind inside the repository, so `git` will not show a stray binary. Run `./install.sh` again after pulling updates.

**Windows**
```shell
go build -o sculk.exe .
```
Move `sculk.exe` to a permanent folder and add that folder to your PATH.

## Quick start

Open a shell *inside* your datapack folder (e.g. `saves/WORLD_NAME/datapacks/my_pack`):
```shell
sculk init --dp <namespace> <gameVersion>   # start a sculk project
sculk add id-system                          # add a library
sculk list                                   # see what is available
```
Not sure about a command? `sculk --help` and `sculk <command> --help` list everything.

---

## Todo

### Goals
- [X] Make a website - Sculk users can find available libraries to download on this website.
- [X] ~~Make showcase gifs/videos~~
- [X] ~~Conflict-less merging of libraries~~
- [X] ~~storing all installed libraries into a `libraries.json` (similar to package.json)~~

### Commands
- [X] Add 'template' command.
    - [X] `sculk template --create [templateName]`
    - [X] `sculk template --delete [templateName]`
    - [X] `sculk template --add [templateName]`
- [X] Add 'config' command.
    - [X] Add option to install modules in a seperate datapack (no merging).
        - [X] `sculk config doMerge true/false`
    - [X] Add default author-name.
        - [X] `sculk config author [authorNameString]`
- [X] Add particular version installation via searching branches 
    - [X] `sculk add id-system@1.0.0` <- 1.0.0 version, current GameVer.
    - [X] `sculk add id-system@1.0.0/26.2` <- 1.0.0 version of 26.2 GameVer.
    - [X] `sculk add id-system@/26.2` <- latest library for 26.2 GameVer.
    - [X] `sculk add id-system@latest` <- same as `sculk add id-system`; if --ignore flag used, downloads from latest GameVer.

- [X] ~~Handle library versioning, updating.~~
- [X] ~~An 'uninstall' command to remove files without causing conflicts.~~
- [X] ~~an 'install' command to install all libraries mentioned in `libraries.json`.~~

### Other
- [X] Add advanced version comparing (e.g. "<1.20.1" or ">26.2" becomes valid syntax).
- [X] Write some default libraries
    - [X] id-system - an id library that creates scoreboard ids
    - [X] uuid - hexuuid conversion library (CJDev's library)
    - [X] astar - CJDev's astar
    - [X] playermotion - https://github.com/MulverineX/player_motion
    - [X] titlewriter - https://github.com/officialbarden/titlewriter
    - [X] speclib - https://github.com/officialbarden/speclib
    - [X] reef - https://github.com/Trioplane/reef
    - [X] hitmatch - https://github.com/picarrow/hit-match
    - [X] stringlib - https://github.com/CMDred/StringLib

 <br>

# Welcome to Sculk-CLI
Sculk CLI is a CLI-app for project initialisation, datapack library* handling, sharing and merging, inspired by Nodejs' Package Manager (NPM), Python's PIP & Rust's Cargo.

## Why should I use sculk-cli?

Similar to npm, pip and cargo, sculk-cli (or just 'sculk') is meant to be a CLI tool that allows datapack developers to seemlessly integrate libraries created by other datapack developers and verified by sculk devs. 

Generally, the process of installing a library* from an external source, like Github, Smithed or Modrinth, takes a while as the developer has to navigate to the library*'s project page, choose the version that is satisfactory to them, click install and drag-drop-extract the zip file in their working directory. Sculk-cli makes it all possible in ~3 command-line tools!

Similarly, because there is no direct Mojang support for what a default datapack can be (except the '/datapack create' command), it gets annoying to create the same #minecraft:load or #minecraft:tick files everytime you want to setup a new datapack project. Sculk fixes that by creating the directories with just one command!


# Initializing a 'sculk project'

Open the shell *inside* your datapack folder (e.g. /saves/WORLD_NAME/datapacks/<test_datapack>)

```shell
sculk init --dp/--rp <namespace> 26.3
```

This will *initialize* a new 'sculk project' inside the datapack.
```
/test_datapack
    ├── data
    │     ├── minecraft/tags/function
    │     │     ├── tick.json
    │     │     └── load.json
    │     └── <namespace>/function/global
    │     │     └── load.mcfunction
    │     │     └── tick.mcfunction
    ├── pack.mcmeta
    └── libraries.json
```

libraries.json is a json file used by **sculk** to find/track/log libraries. During initialization, it'll look something like this:
```json
{
   "author": "USER",
   "version": "1.0.0",
   "game_version": "26.2",
   "libraries": []
}
```
The fields in libraries.json are pretty self explanatory. 

The 'version' field is a semantic way for a sculk developer to versonify their projects / libraries. It also allows sculk-cli to find updated libraries for the same game version, in a sculk-project*.

The 'game_version' field is important, as it tells the sculk-cli which game version is your sculk-project* at the current moment is compatible with.

Notes:
- sculk-project: A sculk project can be both - a **datapack** or a **resourcepack**. Hence, instead of specifying a datapack/resourcepack as is, the docs use the term 'sculk-project'.

# Installing a sculk library*

Just like how a developer may download a particular package, say for example: the pandas library* of python, the command for it in command-line would be: `pip install pandas`

Similarly, sculk-cli uses acronyms/identifiers to identify certain libraries/projects. Sculk doesn't have its own package storage and hosting platform, so it sources the packages from Git Hosting platforms like GitHub and Codeberg. Internally, each unique "identifier" is mapped to a source repository, which is cloned and seemlessly merged with the current project. Alternatively, you can also provide a link directly.

Run the following command:
```
sculk add id-system
```

Sculk will then reach in its internal hashmap, get the source link and get the contents of the library*, and merge them with your sculk-project as is. Your libraries.json will change, and a new library* will be added in the 'libraries[]' field.

```json
{
   "author": "USER",
   "version": "1.0.0",
   "game_version": "26.2",
   "libraries": [
      {
         "identifier": "id-system",
         "source": "http://github.com/officialbarden/id-system",
         "version": "1.0.0",
         "game_version": "26.2"
      }
   ]
}
```

Here, you can see that the library* information is stored inside your `libraries.json` file. You can share this .json file with other sculk developers, and ask them to run the command `sculk install` to install all packages specified in the 'libraries[]' field.

If you're working on an unstable version, like a snapshot: you can use the `--ignore` flag to ignore Game Version Mismatch Checking and install packages directly meant for other Game Versions into your sculk project. 

Alternatively, you can also run the command:
Run the following command:
```
sculk add https://github.com/officialbarden/id-system
```

Sculk will then install the contents of the repository into your datapack, and will update your `libraries.json` to look like the following:

```json
{
   "author": "USER",
   "version": "1.0.0",
   "game_version": "26.2",
   "libraries": [
      {
         "identifier": "http://github.com/officialbarden/id-system",
         "source": "http://github.com/officialbarden/id-system",
         "version": "1.0.0",
         "game_version": "26.2"
      }
   ]
}
```

This allows Sculk to have a *decentralized library ecosystem*.

*Note: By using the --ignore flag, you are to take full responsibility of how the library interacts with your sculk project, as some additional stuff, like tags, advancements, predicates and more may get installed and merged into your project without a warning!*

## Pinning versions: `sculk add name@spec`

Every `sculk add` argument accepts an `@` suffix that pins a version and, optionally, a game version. Both halves are full constraint expressions:

```
sculk add id-system                 # latest for the current GameVer
sculk add id-system@1.0.0           # that version, current GameVer
sculk add id-system@1.0.0/26.2      # that version of that GameVer
sculk add id-system@/26.2           # latest version for that GameVer
sculk add id-system@latest          # newest available
sculk add id-system@^1.0.0          # anything inside 1.x
sculk add id-system@">=1.0.0 <2.0.0"
sculk add id-system@<1.20.1         # see "advanced version comparing" below
```

Versions are read from the library's published branches and tags, so a library publishes one ref per release using the convention `<version>` or `<version>/<gameVer>` (a leading `v` is tolerated on tags).

**Note for publishers:** git stores refs as files, so a repository cannot hold both `refs/heads/1.0.0` and `refs/heads/1.0.0/26.2` at once. sculk therefore also accepts the flat spellings `1.0.0+26.2` and `1.0.0_26.2` for the qualified refs, which never collide with a plain `1.0.0` branch. A hyphen is not a separator, because game versions contain one (`26.4-snapshot-1`).

## `sculk config`

`sculk config` with no arguments prints every value; `sculk config <param> <value>` changes one.

```
sculk config                        # show everything
sculk config doMerge false          # install as separate packs instead of merging
sculk config author Barden          # default author for new libraries.json
sculk config initTemplate none      # template used by 'sculk init'
sculk config reset                  # rewrite the defaults
```

With `doMerge false`, `sculk add` lays the library down as its own datapack next to your project (in the world's `datapacks/` folder), which Minecraft loads separately. This is the recommended mode for whole engines such as `macroengine`. Resourcepack libraries are redirected to the sibling `resourcepacks/` folder. If your project is not inside a `datapacks/` folder, sculk installs into `<project>/libraries/<id>` instead and tells you where to move it, rather than writing outside your project.

## `sculk list`

```
sculk list                          # everything sculk knows about
sculk list string                   # narrow by keyword
sculk list --installed              # only what is in this project
sculk list --json                   # machine readable; feeds the website
```

## Advanced version comparing

The game-version compatibility check now understands the full constraint syntax, so a library may declare a range instead of one exact version in its `libraries.json`:

```
"game_version": ">=26.2"        # works on 26.2 and newer
"game_version": "<1.20.1"       # older than 1.20.1
"game_version": ">=1.20 <26.0"  # AND
"game_version": "1.0.0 || 2.0.0" # OR
```

The same syntax is accepted on the command line (`sculk add id-system@<1.20.1`). Supported operators: `=`, `==`, `!=`, `<`, `<=`, `>`, `>=`, `^` (caret), `~` (tilde) and `x`/`*` wildcards; ranges combine with `,`/whitespace (AND) and `||` (OR).

## macroEngine

`sculk add macroengine` installs the string / cooldown / multi-command framework from `IronCrest-sudo/core` (`archived/macroEngine-Datapack-v26.4`). It ships a load gate, so after installing it stays inert until an operator runs `function macroengine:gate/v26_4/confirm {format:122}`. Because it targets Minecraft `26.4-snapshot-1`, install it with `--ignore` on other game versions, and prefer `sculk config doMerge false`. Its companion `macroengine-rp` provides the engine text, sounds and trim assets.

## Why have a seperate libraries.json if the library* contents are merged in the final datapack, anyways?

Great Question! Sculk isn't just a 'library* installer & merger', it's a full blown package manager! (well, a barebones version of a traditional package manager). By having a seperate 'libraries.json' file in the sculk-project's root directory, sculk developers can keep track of what libraries exist in their codebase! At the same time, say a sculk-library* developer pushed a new version of the library* onto their git repository, by running one command in the shell (`sculk update`), sculk developers can pull the changes into their files without the hassle of merging new updates by hand.


Notes:
- library: in npm & pip, they're called packages. In cargo, they're called crates and in sculk, they're called libraries. Though, we don't object to interchangability of the two terms: library and package.

# Creating/Publishing a sculk library
You can create and submit your sculk libraries in our [sculk-discord](
https://discord.gg/JWZkAgsyry) for approval and direct integration into the cli. As long as your datapack outputs into a standard datapack, you may use whatever pre-compiler you desire however, for seemless integration with sculk, here are some rules you will have to follow while writing your sculk library.

1. Use a unique namespace<br>
Generally, we'll encourage library publishers to use the same as the identifier they wish to associate their sculk library with. However, incase there are issues like the identifier slug being pre-used, we ask developers to change their namespaces to a much more customized and unique slug. You can look at the [wiki-guide](https://minecraft.wiki/w/Identifier#Legal_characters) to learn what characters you can add to your namespace to make it more unique. <br><br>
As best practice, we would encourage developers to follow [Smithed's Conventions](https://docs.smithed.dev/conventions/index.html).

2. Do Not Use Extremely Minimal Namespaces/Filenames <br>
Same as the first rule, to prevent potential merge conflicting with other libraries, we hope that your naming convention, especially if you're writing to namespaces that are extremely common (namespaces like 'minecraft', 'code', 'namespace' etc.) for whatever reason, your file name must be unique.

Once you've followed the above recommendations/rules and created a library, make a `libraries.json` and paste the following code inside it:
```json
{
   "author": "AUTHOR-NAME",
   "version": "1.0.0",
   "game_version": "26.2",
   "libraries": []
}
```

Fields like author are currently only semantic. The 'version' field and 'game_version' are what you should pay attention to, as sculk will look at these values while making decisions during the merging of your library.
