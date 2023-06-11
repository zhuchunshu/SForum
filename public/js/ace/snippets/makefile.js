ace.define("ace/snippets/makefile",["require","exports","module"],function(e,t){"use strict";t.snippetText=`snippet ifeq
	ifeq (\${1:cond0},\${2:cond1})
		\${3:code}
	endif
`,t.scope="makefile"}),function(){ace.require(["ace/snippets/makefile"],function(e){typeof module=="object"&&typeof exports=="object"&&module&&(module.exports=e)})}()