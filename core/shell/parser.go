package shell

// Defined by
// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html

/**
1. The shell reads its input from a file (see sh), from the -c option or from
the system() and popen() functions defined in the System Interfaces volume of
POSIX.1-2017. If the first line of a file of shell commands starts with the
characters "#!", the results are unspecified.

2. The shell breaks the input into tokens: words and operators; see Token Recognition.

3. The shell parses the input into simple commands (see Simple Commands) and
compound commands (see Compound Commands).

4. The shell performs various expansions (separately) on different parts of each
command, resulting in a list of pathnames and fields to be treated as a command
and arguments; see wordexp.

5. The shell performs redirection (see Redirection) and removes redirection
operators and their operands from the parameter list.

6. The shell executes a function (see Function Definition Command), built-in
(see Special Built-In Utilities), executable file, or script, giving the names
of the arguments as positional parameters numbered 1 to n, and the name of the
command (or in the case of a function within a script, the name of the script)
as the positional parameter numbered 0 (see Command Search and Execution).

7. The shell optionally waits for the command to complete and collects the exit
status (see Exit Status for Commands).
**/
