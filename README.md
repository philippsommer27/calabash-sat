# calabash-sat
Calabash SAT is a tool to analyze green patterns within a collection of projects. Using semgrep as a rule matching engine, it allows users to provide a set of rules, and grade multiple projects against each other.

## Download and Install

1. **Go to the Releases Page**:
   - Navigate to the [Releases page](https://github.com/philippsommer27/calabash-sat/releases).

2. **Download the Appropriate Binary**:
   - Download the binary for your operating system:
     - `calabash-sat_<version>_windows_amd64.zip` for Windows
     - `calabash-sat_<version>_linux_amd64.tar.gz` for Linux
     - `calabash-sat_<version>_darwin_amd64.tar.gz` for macOS

3. **Extract the Binary**:
   - For Windows:
     - Right-click the downloaded `.zip` file and select "Extract All".
   - For Linux/macOS:
     - Use the terminal to extract the `.tar.gz` file:
       ```bash
       tar -xzf calabash-sat_<version>_linux_amd64.tar.gz
       ```

4. **Add the Binary to PATH** (Optional):
   - **Windows**:
     - Copy the extracted `calabash-sat.exe` to a directory of your choice (e.g., `C:\Program Files\CalabashSAT`).
     - Add this directory to your PATH:
       - Open the Start Menu, search for "Environment Variables", and select "Edit the system environment variables".
       - In the System Properties window, click on the "Environment Variables" button.
       - In the Environment Variables window, find the "Path" variable in the "System variables" section and select it.
       - Click "Edit", then "New", and add the path to the directory where you copied the binary (e.g., `C:\Program Files\CalabashSAT`).
       - Click "OK" on all windows to apply the changes.
   - **Linux/macOS**:
     - Move the binary to `/usr/local/bin`:
       ```bash
       sudo mv calabash-sat /usr/local/bin/
       ```
     - Ensure the binary is executable:
       ```bash
       sudo chmod +x /usr/local/bin/calabash-sat
       ```

5. **Run the CLI Application**:
   - Open a terminal.
   - Run the application:
     ```bash
     calabash-sat --help
     ```
# Usage
Using this tool involves two phases, corresponding to the two commands available.

## Analyze rules within a set of projects.
You must use the tool to determine the prevelance of projects in your project dataset. To do this use the following command: `evalrule <path to rule directory> <path to projects directory> <path output directory>`

You may additionally use the `-P` tag to print semgreps output or `-M` to enable multithreading.

You should execute this command for each rule in your rules set.

## Produce overall grades
The second function allows you to calculate an overall grade for each project based on the individual pattern grades and a mapping of pattern to severity. The mapping allows you to make certain patterns affect the overall grade more than others.

First place all your result files from the first command for each pattern into one folder. Then create a json file that maps the pattern to a severity weight between 1 and 3.

Finally run the command `evalprojs <path to directory containing findings>`
