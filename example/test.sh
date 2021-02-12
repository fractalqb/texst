#!/bin/sh
TEXST=../cmd/texst/texst
$TEXST -r reference.texst subject
$TEXST -r README.log.texst README-reference.log README-subject.log