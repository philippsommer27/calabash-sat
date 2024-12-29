# calabash-sat
Calabash SAT is a tool to analyze green patterns within a collection of projects. Using semgrep as a rule matching engine, it allows users to provide a set of rules, and grade multiple projects against each other.

## Installation

0. **Install semgrep cli**:
    - On Mac this is most easily done throw homebrew: `brew install semgrep`
    - For others platforms, see [semgrep README](https://github.com/semgrep/semgrep?tab=readme-ov-file#option-2-getting-started-from-the-cli)

1. [**Install Go**](https://go.dev/doc/install)

2. **Install Calabash SAT**
    - `go install github.com/philippsommer27/calabash-sat@latest`

5. **Run the CLI Application**:
   - Open a terminal.
   - Run the application:
     ```bash
     calabash-sat --help
     ```
# Usage
Using this tool involves two phases, corresponding to the two commands available.

## Analyze rules within a set of projects.
You must use the tool to determine the prevelance of projects in your project dataset. To do this use the following command: `evalrule <path to rule directory> <path to projects directory> <path output directory> <language>`

You may additionally use the `-P` tag to print semgreps output or `-M` to enable multithreading.

You should execute this command for each rule in your rules set.

## Produce overall grades
The second function allows you to calculate an overall grade for each project based on the individual pattern grades and a mapping of pattern to severity. The mapping allows you to make certain patterns affect the overall grade more than others.

First place all your result files from the first command for each pattern into one folder. Then create a json file that maps the pattern to a severity weight between 1 and 3.

Finally run the command `evalprojs <path to severity ratings> <path to directory containing findings>`
For an example of what the severity ratings should look like, see [here](https://github.com/philippsommer27/calabash-catalog/blob/main/catalog/static-analysis/Overall%20Results/testInfo.json)
