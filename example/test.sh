#!/bin/sh
TEXST=../cmd/texst/texst
$TEXST compare -r reference.texst subject
$TEXST compare -r README.log.texst README-reference.log README-subject.log
