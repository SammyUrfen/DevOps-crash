# Docker, in plain words

No jargon. Where a word is unavoidable, it gets explained the first time.

---

## 1. What problem Docker solves for you

Your program is already one file that runs on any Linux computer. That part is done.

But your program does not work alone. It needs a database. The database needs a set version, a place to keep the data, and it has to start before your program looks for it. Setting that up by hand, on every computer, is the work. Docker removes that work.

---

## 2. The one idea: a sealed lunchbox

Think of a lunchbox.

You **pack** it one time. Inside goes your program, and the few small things your program opens while it runs. The lunchbox is now sealed. You can copy it to another computer and it is identical, down to the last crumb. Docker calls this an **image**.

You **open** the lunchbox to eat. The lunchbox itself stays sealed and unchanged, so you can open ten copies at the same time. Docker calls one opened copy a **container**.

When you finish, the wrappers go in the bin. Anything your program wrote while it ran is gone. That is on purpose, and it is the part that surprises people.

---

## 3. Your program gets a fake computer

When Docker runs your program, it hands the program a small pretend computer:

- Its own set of files. Only what you packed. Nothing from your laptop.
- Its own network card, with its own idea of "this computer".
- Its own list of running programs. Yours is the only one.
- A cap on how much memory and processor it may use.

Your program cannot tell the difference. It believes it owns a whole machine.

**One sentence explains most confusing errors: "this computer" inside the box is the box, never your laptop.**

That is why your first run failed. Your program looked for the database on "this computer", searched its own pretend machine, found nothing, and gave up after 30 seconds.

---

## 4. Why the database gets its own box

Your program is throwaway. You rebuild it twenty times a day.

The database holds the tasks that people typed. That has to survive.

So they are two separate boxes. The database box also gets a named storage area that lives outside the box. Throw the box away and the tasks stay. That storage area is the `pgdata` line in `docker-compose.yml`.

Rule: a box is disposable. Anything worth keeping lives outside it.

---

## 5. How the two boxes talk

Put both boxes on the same private network, and Docker gives each box a name on that network.

Your program asks for a computer called `db`. Docker answers with the database box. You never write down an address, and the address can change without breaking anything.

Forget the network and your program falls back to the rule in section 3. It looks on its own pretend computer, and nothing is there.

---

## 6. Two things you hand in from outside

You do not pack everything. Two things arrive when the box starts.

**Settings.** Where the database is, which door to listen on, how much to write to the screen. You pass these in each time. One lunchbox, different settings, different place. This is why the same sealed image works on your laptop and in production.

**A door.** By default nothing outside can reach your program. You open exactly one door, and you say which door on your laptop maps to which door inside the box. On your machine another program already sits on door 8080, so the examples here use door 8081 on the laptop, and door 8080 inside the box.

---

## 7. The two recipes in this project

**The pack list**, named `Dockerfile`. It has two halves.

The first half is a full kitchen. It holds the Go compiler, and it weighs 1.25 GB. It cooks your program.

The second half is the lunchbox. It takes the cooked program and throws the whole kitchen away. What is left is your program plus three small things:

| Small thing | Why your program needs it |
|---|---|
| A list of trusted signatures | Later your database moves to the internet. Your program has to check that the server answering is the real one, and not somebody pretending. It checks against this list |
| A name for a user | So your program does not run as the all-powerful administrator inside the box. If someone breaks in, they get a weak account |
| Timezone names | So a date like "Asia/Kolkata" means something. Your program does not use this yet |

Nothing else. No shell, no text editor, no tools at all. That is deliberate, and section 9 covers both sides of it.

**The do-not-pack list**, named `.dockerignore`. Before Docker packs anything, it copies the whole project folder to itself. Without this list it copied 31 MB of old junk every single time. With it, 3.5 kB.

---

## 8. Why bother, given your program is one file

Three honest reasons. Nothing about "it works on my machine", because Go already fixed that.

1. **The database.** One command brings up the exact right version, with its storage, ready to answer.
2. **One name for one exact thing.** Every sealed lunchbox gets a long fingerprint. You can prove the box you tested and the box that is running hold the same bytes. A loose program file plus a start script plus a settings file is three things with no shared fingerprint.
3. **Everything later needs it.** Kubernetes, Helm, Argo CD and Render all take a lunchbox as input. None of them take a loose program. This is the step that connects to the rest of the course.

---

## 9. What Docker makes worse

| Harder | Why |
|---|---|
| Addresses | Every address gains one step. "This computer" no longer means what you think |
| Looking inside | The lunchbox holds no tools, so you cannot open a prompt inside it. You read what the program printed. The same emptiness stops an intruder from doing anything |
| Keeping things | Anything written inside disappears, unless you set up storage outside first |
| Asking for "the newest" | "Newest" is a sticker, and a sticker can be moved to a different box. Two computers can get different things under one name. The long fingerprint never moves |
| Waiting to stop | Docker asks your program to stop, waits 10 seconds, then kills it. Your program asks for 15 seconds to finish open requests. It never gets them |

---

## 10. The commands you will use

### Pack a lunchbox

```bash
docker build -t tasks:slim .
```

| Part | Plain meaning |
|---|---|
| `build` | Follow the pack list and seal a lunchbox |
| `-t tasks:slim` | Put the sticker `tasks:slim` on it. Without a sticker you get a long fingerprint to copy by hand |
| `.` | The folder to copy across before packing. Not "pack this folder" |
| `--no-cache` | Redo every step. Docker reuses finished steps by default. Use this only when you suspect a stale step |
| `--progress=plain` | Print everything each step says. This is how you read a build that failed |

Docker reuses a step when nothing that feeds it changed. This is why the pack list asks for the dependency list first and your code second. Change one line of Go and the dependency download does not run again.

### Open one

```bash
docker run -d --name tasks \
  --network devops-crash_default \
  -e DATABASE_URL='postgres://devops:devops@db:5432/devops?sslmode=disable' \
  -p 8081:8080 \
  tasks:slim
```

| Part | Plain meaning |
|---|---|
| `run` | Open a lunchbox and start the program inside |
| `-d` | Leave it running in the background and give me my prompt back |
| `--rm` | Throw the box away the moment the program stops |
| `--name tasks` | Call this open box `tasks`, so later commands can name it |
| `--network devops-crash_default` | Join the private network that the database is already on |
| `-e KEY=value` | Hand in one setting |
| `--env-file .env` | Hand in many settings from one list |
| `-p 8081:8080` | Open a door. Left is the door on your laptop. Right is the door inside the box |
| `-v pgdata:/var/lib/postgresql/data` | Attach outside storage at that spot, so what is written there survives |
| `--entrypoint ls` | Run a different program instead of the packed one |

**The trap that caught you.** Your lunchbox is packed to run `/server`. Extra words you type become instructions **to** `/server`, not a replacement for it. `docker run tasks:slim ls` runs `/server ls`. To run something else, use `--entrypoint`.

### Watch it

| Command | Plain meaning |
|---|---|
| `docker ps` | Which boxes are open. Add `-a` to include closed ones |
| `docker logs tasks` | Everything the program printed. `-f` keeps printing as it goes, `--tail 20` shows the last 20 lines |
| `docker images` | Which lunchboxes you have |
| `docker stats` | Live memory and processor use per box |
| `docker network ls` | Which private networks exist |

A program in a box prints to the screen and never writes a log file. `docker logs` replays that screen output. This is the only window you get.

### Stop and tidy

| Command | Plain meaning |
|---|---|
| `docker stop tasks` | Ask the program to finish, wait 10 seconds, then kill it. `-t 20` waits 20 |
| `docker start tasks` | Open it again with the same settings |
| `docker rm tasks` | Throw away a closed box. `-f` closes and throws away in one go |
| `docker rmi tasks:slim` | Delete a lunchbox |
| `docker builder prune` | Free the space Docker keeps for reusing steps. Usually the biggest win |
| `docker system prune -a` | Delete everything unused. Read the warning before you agree |

### Several boxes at once

One file, `docker-compose.yml`, describes several boxes and starts them together. It also creates the private network, named after the folder plus `_default`. Here that is `devops-crash_default`.

| Command | Plain meaning |
|---|---|
| `docker compose up -d` | Start everything in the file, in the background |
| `docker compose down` | Stop and remove the boxes and the network. Outside storage stays |
| `docker compose down -v` | Also delete the outside storage. **This erases every task in the database** |
| `docker compose ps` | Status of this project only |
| `docker compose logs -f db` | Watch one of them |
| `docker compose exec db psql -U devops -d devops` | Run something inside a box that is already open |

### Hand a lunchbox to GitHub

```bash
echo "$GH_TOKEN" | docker login ghcr.io -u SammyUrfen --password-stdin
docker tag tasks:slim ghcr.io/sammyurfen/devops-crash:v1
docker push ghcr.io/sammyurfen/devops-crash:v1
```

| Part | Plain meaning |
|---|---|
| `login` | Prove who you are to the storage service |
| `tag` | Add a second sticker that says where it belongs |
| `push` | Upload it |
| `ghcr.io` | GitHub's storage for lunchboxes. Leave the name out and Docker uses Docker Hub |
| `v1` | Your version sticker. Avoid `latest`, because that sticker moves |

---

## 11. Copy and paste

Build it, start the database, open the box, check it works:

```bash
docker build -t tasks:slim .
docker compose up -d
docker run -d --name tasks \
  --network devops-crash_default \
  -e DATABASE_URL='postgres://devops:devops@db:5432/devops?sslmode=disable' \
  -p 8081:8080 tasks:slim
./smoke.sh http://localhost:8081
```

I ran this. All 14 checks passed.

Watch it, then stop it:

```bash
docker logs -f tasks
docker stop -t 20 tasks && docker rm tasks
```

Start over from nothing:

```bash
docker rm -f tasks
docker compose down -v      # this erases the database
docker builder prune -f
```

---

## 12. One thing left to fix

`internal/config/config.go` asks for 15 seconds to finish open requests. `docker stop` waits 10 and then kills. Pick one of three fixes:

1. Always type `docker stop -t 20 tasks`.
2. Add `stop_grace_period: 20s` to `docker-compose.yml`.
3. Lower the 15 seconds in the Go code to under 10.

Kubernetes has the same setting, called `terminationGracePeriodSeconds`, and it waits 30 seconds by default.
