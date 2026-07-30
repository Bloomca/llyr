# Llŷr

Llŷr is a CLI application to do GitHub PR reviews using your local agent. It is designed mainly to provide self-reviews and allow to follow-up on the feedback, so you can build your own projects with some sanity checks.

## Installation

Currently, the only way to install it is using Go. After you [installed](https://go.dev/doc/install) the language, run:

```sh
go install github.com/bloomca/llyr@latest
```

This will place it into the Go binary folder, which _should_ be already in your path. After that, simply run `llyr`, it will ask you to select an agent. After that, 2 main commands are `llyr review %PR_URL%` and `llyr reply %PR_URL%`.

You'll also need [gh CLI](https://cli.github.com/). This tool works only with GitHub and requires an authenticated `gh` session.

## Usage

This tool provides PR reviews and allows you to communicate with it. So why use this tool instead of a skill inside your agent? The main reason is to decouple the implementation and review, and to allow communication on specific feedback. This tool allows you to comment on the feedback and then receive replies which will be specific to that conversation.

Since it mostly aimed at self-reviews, it uses your own GH identity in the `gh` CLI tool, and prepends `> Posted by Llŷr` to each review/comment it makes. It will also only work on repositories where you are the admin.

In order to use it, you need to configure a local agent to use. Currently the tool has parameters for 3 options: [pi](https://github.com/earendil-works/pi/tree/main/packages/coding-agent), [claude code](https://code.claude.com/docs/en/overview) and [codex](https://github.com/openai/codex). You can provide your own CLI tool with all the relevant arguments (prompt will be as a last argument during invokation). You can run `llyr config` to change the agent tool.

After that, you have 2 main commands:

- `llyr review https://github.com/Bloomca/llyr/pull/27`
- `llyr reply https://github.com/Bloomca/llyr/pull/27`

The first command will pull the latest version of that PR and review it using the selected agent tool and post the review there. The second command will look for any responses to the feedback you left and answer in each conversation.

## Model

One caveat is that if you use a model with maximum reasoning, it will take quite some time, especially if the PR is decently sized or just touches many components. If you select any default option for the agent tool, it will reuse your select model and reasoning level from it, so you can experiment with installing a separate agent for that only. You can also select a specific mode in the custom agent tool command, for example:

```
codex -a never -s read-only -m gpt-5.6-terra -c 'model_reasoning_effort="high"' exec
claude -p --permission-mode auto --model sonnet --effort high
```

> If you provide your own agent tool command, make sure it is an auto permission mode and sandboxed

## Security

Since this tool is aimed at self-reviews, intended to be run manually and assumes trusted environment, it does not have any sandbox mechanism. The logic to fetch the repository, conversations, checking the owner of the repo, validating feedback author and posting the review are all done deterministically inside the Go app, but the review itself happens in the agent tool, so you need to be aware of the prompt injection attacks, so you should not run it for untrusted PRs.

Claude and Codex by default have their own sandbox mechanisms, but Pi does not.

## Full commands list

```text
llyr review <pull-request-url>  Review a pull request
llyr reply <pull-request-url>   Answer replies to the latest Llŷr review
llyr config                     Change the configured agent command
llyr clean                      Remove all cloned repositories
llyr version                    Show version
llyr help                       Show command help
```

## License

MIT