# y2bnf

conver yacc to BNF for markdown renderer

```
xhe@M1PRO y2bnf % ./y2bnf -in parser.y -include Identifier -exclude ""
Identifier ::=
	 identifier
	 | UnReservedKeyword
	 | NotKeywordToken
	 | TiDBKeyword

xhe@M1PRO y2bnf % ./y2bnf -in parser.y -include Identifier -exclude "" --recursive
Identifier ::=
	 identifier
	 | UnReservedKeyword
	 | NotKeywordToken
	 | TiDBKeyword

UnReservedKeyword ::=
	 "ACTION"
	 | "ADVISE"
```
