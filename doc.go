/*
Package texst compares text files against a reference text
specifications. A specification consists of the reference text itself
combined with options how the reference text is matched. Lines with
reference text are marked with a prefix that has '>' in the first
column. The simplest reference text would be the verbatim text with
each line prefixed with a "> ".  This would only match exactly the
verbatim text. To do more complex matching one can add other line
types to the reference text specification.  Line types are recognized
by the rune in the first column of each line in the reference text
specification. There are line types that serve different purposes.

Most often one might need to mark parts of a reference line that do
not need to match exactly the compared “subject” text. We will call
these markers 'masks'.  Each reference text line may be followed by
argument lines, that define masks and the way the reference text is
matched against them. Argument lines start with ' ' (U+0020). There
are different types of argument lines, e.g. this one starting with "
.":

	> This is some reference text content
	 .        xxxx

The above example says that the four runes above the non-space part of
the argument line, i.e. "some", are not compared to the compared
subject text. The second column, here '.', identifies the specific
type of argument line, for details see Types of Argument Lines. The
text

	This is blue reference text content

would perfectly match the reference text example. Argument lines can
be stacked and are applied in order to their reference text line up to
the next non-argument line.

	> This is some reference text content
	 .        xxxx
	 .                       yyyy

would be the same as

	> This is some reference text content
	 .        xxxx           yyyy

For some files, e.g. log files, it would be rather tedious if one had
to mark each timestamp in the reference text line:

	Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
	Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
	Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
	…

To solve this one can set a global mask line in the preamble and. For
our example one would write:

	*.ttt tt tt tt tt ttt
	> Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
	> Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
	> Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
	> …

With a little attention, you notice that the log lines are from
different threads. I.e. one cannot rely on the order of lines in the
reference text specification. But at least the lines from one thread
shall be in exactly the same order as given in the reference.

For this we declare two “interleaving groups” '1' and '2' in the
preamble and mark the reference text lines to be in the specific
group:

	%%12
	*.ttt tt tt tt tt ttt
	>1Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
	>2Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
	>1Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
	> …

Now, both subject texts

	Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
	Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
	Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
	…

and

	Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
	Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
	Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
	…

match the reference text.

# Comparing Subject and Reference

Comparing subject texts is done by scanning the subject text line by
line and then matching the current line against the reference lines
currently in question. For each Interleaving Group there is at most
one reference line in question. The first successful match of the
subject line with a reference line accepts the subject line. Then the
matched reference text line is replaced with the next reference text
line from the same interleaving group, if any.  Afterward scanning
continues with the next subject line. Reference lines from different
interleaving groups are checked in the same order as they are declared
in the preamble.

If the subject line does not match, a mismatch is reported and
scanning continues with the next subject text line. One can configure
a maximum number of mismatches that is processed before scanning is
aborted. By default the complete subject text is scanned.

# Types of Argument Lines

TODO: Be more descriptive

There are the following types of mask lines:

	. Part of subject must have same length as mask
	* Part of subject may be of any length, even 0
	+ Part of subject may be of any length >0
	0 Part of subject may be of any length up to the length of the mask
	1 Part of subject may be of any length >0 up to the length of the mask
	- Part of subject must be at least as long as the mask

# Preamble Lines

The preamble ends with the first reference line. The type of a
preamble line is recognized from the rune in the first column of the
line, e.g.:

	%%<interleaving groups>

This is a preamble line with tag '%' that sets the interleaving groups
of the reference text specification.

A global mask template can be defined with the preamble line type '*'
similar as the masks of a reference line are defined:

	*.xxx yyy
	*-        zzzzz

# Interleaving Groups

Interleaving groups are identified by a single rune and have to be
declared upfront in the preamble. If no interleaving group is declared
then the interleaving group ' ' (U+0020) is defined by default. A
reference text line is assigned to an interleaving group by the rune
in the second column of the line.  E.g. the lines

	%% a
	> 1st reference text line
	>a2nd reference text line

put the reference text "1st reference text line" into the interleaving
group ' ' and the reference text "2nd reference text line" into the
interleaving group 'a'. Because not only the default group ' ' is
used, the groups had to be declared in the preamble line.

Interleaving groups are defined once in the preamble in a line
starting with '%%'.

TODO: What do these groups do (see "Matching Reference Lines")? => Ambiguities &
Order of IGroups
*/
package texst
