# texst – Text Tests
Package texst checks text files against a reference text
specifications. The simplest reference text would be the verbatim text
prefixed with a 'reference text' line tag, e.g. "> ". This would only
match exactly the verbatim text. To do more complex matching one can
add other line types to the reference text specification.

Line types are recognised by the rune in the first column of each line
in the reference text specification. There are line types that serve
different purposes.

Most often one might need to mark parts of a reference line that do
not need to match exactly to the checked “subject” text. texst does
not embed markers into the reference text line because it would need
some very sophisticated escaping to make arbitrary reference text
feasible.  Instead each reference text line may be followed by
argument lines, that modify the way the reference text is matched
against the checked text. Argument lines start with ' ' (U+0020). Some
types of argument lines are used to mark segments of the reference
text to not match exactly to the subject text:

```
> This is some reference text content
 =        xxxx
```

The above example says that the four runes above the non-space part of
the argument line, i.e. "some", are not compared to the checked
text. The '=' identifies the specific type of argument line (see Types
of argument lines). So the text

```
This is blue reference text content
```

would perfectly match the reference text example. Argument lines can
be stacked and are applied in order to their reference text line up to
the next non-argument line.

```
> This is some reference text content
 =        xxxx
 =                       yyyy
```

would be the same as

```
> This is some reference text content
 =        xxxx           yyyy
```

For some files, e.g. log files, it would be rather tedious if one had
to mark each timestamp in the reference text line:

```
Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
…
```

To solve this one can set a global segment line after the preamble and
between reference text specifications. For our example one would
write:

```
*=ttt tt tt tt tt ttt
> Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
> Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
> Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
> …
```

With a little attention, you notice that the log lines are from
different threads. I.e. one cannot rely on the order of lines in the
reference text specification. But at least the lines from one thread
shall be in exactly the same order as given in the reference.

We declare two “interleaving groups” '1' and '2' in the preamble and
mark the reference text lines to be in the specific group:

```
\%12
*=ttt tt tt tt tt ttt
>1Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
>2Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
>1Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
> …
```

Now, both subjects

```
Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
…
```

and

```
Jun 27 21:58:11.112 INFO  [thread1] create `localization dir:test1/test.xCuf/l10n`
Jun 27 18:58:11.125 DEBUG [thread1] clearing maps
Jun 27 21:58:11.113 INFO  [thread2] load state from `file:test1/test.xCuf/bcplus.json`
…
```

match the reference.


## Preamble Lines

The type of a preamble line is recognized from the rune in the second
column of the line:

```
\%<interleaving groups>
```

A preamble line with tag '%' sets the interleaving groups of the
reference text specification.
