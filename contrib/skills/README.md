# Unleash skills (public stub)

The full skills + operator-instruction pack lives in the **private** repo:

**https://github.com/NetVar1337/unleash-skills**

## Public builds
`unleash install-skills` expects embedded `contrib/skills`. This stub keeps
the embed path valid. To ship skills in a private build:

1. Clone `NetVar1337/unleash-skills`
2. Copy `contrib/skills` (and optionally `contrib/rules`) into this tree
3. Mirror into `go/embed/contrib/skills` before `go build`

Or run install from the private pack checkout.
