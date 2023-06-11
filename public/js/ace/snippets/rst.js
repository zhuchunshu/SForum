ace.define("ace/snippets/rst",["require","exports","module"],function(e,t){"use strict";t.snippetText=`# rst

snippet :
	:\${1:field name}: \${2:field body}
snippet *
	*\${1:Emphasis}*
snippet **
	**\${1:Strong emphasis}**
snippet _
	\\\`\${1:hyperlink-name}\\\`_
	.. _\\\`$1\\\`: \${2:link-block}
snippet =
	\${1:Title}
	=====\${2:=}
	\${3}
snippet -
	\${1:Title}
	-----\${2:-}
	\${3}
snippet cont:
	.. contents::
	
`,t.scope="rst"}),function(){ace.require(["ace/snippets/rst"],function(e){typeof module=="object"&&typeof exports=="object"&&module&&(module.exports=e)})}()