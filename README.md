# Dragon Atelier

```
                                                                       :
                                                                     :+
                                                                    #-
                                                                  %%          -*
                                                               =@%         :#%
                                                            -%@#: =      +@%:
                                                         -%@@++%#     +@@%:
                                                     :#@@@@@@#    :*@@%-
                                                 =#@@@%+-     -#@@@#: :-=
                                             -#@@@%=     =#@@@@@@@%@#=
                                           #=      -#@@@@@@@@@@*:
                                          :-+%@@@@@@%%#+-
                                      +@@@@@@#+-       -+ %@@@@%*=:
                                   +@@@@%- :=*###%%%@@@# :@@@@@@@@@@@#=-
                                 *@@@% #@@@@@@@@@@@@@@:
                               :@@@@%@@@@@@@@@@@@@%=  --  :#@@@*-
                              +@@@@@@@@@@@%#*+-:   +@@@@@%-  +@@@@@+
                             *@@@@@#- :=+=:     :=%@@@@@@@@@=  =@@@@@#
                            +@@@#:#@@@@@@@@@@%+ +#:  :#@@@@@@@:  #@@@@@#
                           :@@# =@@@@@@*       - =@@%-  -%@@@@@#  -@@@@@@-
                          :@@=  @@@@@=   -@@@@*   +@@@@+  :%@@@@%   @@@@@@#
                         :@@- :@@@@@-   #@@@@@@@:  @- @@@=  -@@@@@   @@-%@@@
                        *@@%%@@@@@@#   +@@@@@@@@=  *   =@@%   %@@@@  -@@%  :##
                     -#@@@@@@@@@@@@+   *@@@@@@@@-       :@@@:  #@@@@  +@@@:
                 @@@@@@@@@@@@@@+:       #@@@@@@%          @@@   %@@@=  @@@%
                -@@@-%@@@@@@@@%          :@@@@#           =@@%  :@@@@  =@@@*
                -@@@@@@@@@@# +%@#:   :=%@@@@#              %@@*  +@@@- :@@@@:
                :@@@@@@@## :     =%@@@@@%+:                #@@@  :@@@+  @@@@#
                 %@%=@+:-        *@@*:                     *@#@-  @@@*  @@@@%
                              :@%:                         ** @+  %@@*  @@+#@#
                            :#+                            +- @#  @@@+ :@@@  --
                           :=                              :  @*  @@@= +@@@:
                                                             :@- -@@@  @@@@
                                                             #@: *@@+ +@@@#
                                                            -@+ -@@@ -@@@@-
                                                            %%  %@@  @@=@%
                                                           #%  #@@  @@# @*
                                                          ##  %@% -@@@:  +
                                                         %= :@@+ =@@@-
                                                       =*  *@#   @@@:
                                                     :-  +@+    =@+
                                                      :+=       *-
```

> Mission Control for AI Workflows. A personal Go + Bubble Tea TUI atop the Gentleman stack.

## Status

**Alpha — under construction.** Core scaffold in place; features are coming soon.

## What is Dragon Atelier

Dragon Atelier is a terminal TUI (Text User Interface) that serves as Mission Control for AI-assisted development workflows. Built on the Gentleman stack using Go and Bubble Tea.

Upcoming features:
- Engram memory client — browse and search persistent AI session memory
- Cost tracker — monitor AI API usage and spending
- Hook installer — manage development workflow automation hooks

## Install

### Prerequisites

Install the [Task](https://taskfile.dev) runner (used for all build commands):

```sh
go install github.com/go-task/task/v3/cmd/task@latest
```

### Build from source

```sh
git clone https://github.com/GastonZ/Atelier.git
cd Atelier
task build
```

**Fallback** (without Task):

```sh
go build -o atelier ./cmd/atelier
```

## Run

```sh
./atelier          # Launch the TUI (Linux/macOS)
./atelier.exe      # Launch the TUI (Windows)
```

**Subcommands:**

```sh
atelier version    # Print version and exit
atelier help       # Show usage information
```

Press `q` to quit the TUI.

> Note: terminals smaller than ~100 columns x ~48 rows display a text-only fallback.
> Resize your terminal for the full dragon art experience.

## Project Status

This is an alpha release. The `.goreleaser.yaml` is a provisional stub — it will be validated before the first `v0.1.0` tag.

Upcoming changes planned:
- `engram-client` — TUI interface for Engram memory
- `cost-tracker` — AI API cost monitoring
- `hook-installer` — Development hook management

## License

[MIT](LICENSE) — Copyright 2026 Gaston Zappulla
