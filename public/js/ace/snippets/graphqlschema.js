ace.define("ace/snippets/graphqlschema",["require","exports","module"],function(e,t){"use strict";t.snippetText=`# Type Snippet
trigger type
snippet type
	type \${1:type_name} {
		\${2:type_siblings}
	}

# Input Snippet
trigger input
snippet input
	input \${1:input_name} {
		\${2:input_siblings}
	}

# Interface Snippet
trigger interface
snippet interface
	interface \${1:interface_name} {
		\${2:interface_siblings}
	}

# Interface Snippet
trigger union
snippet union
	union \${1:union_name} = \${2:type} | \${3: type}

# Enum Snippet
trigger enum
snippet enum
	enum \${1:enum_name} {
		\${2:enum_siblings}
	}
`,t.scope="graphqlschema"}),function(){ace.require(["ace/snippets/graphqlschema"],function(e){typeof module=="object"&&typeof exports=="object"&&module&&(module.exports=e)})}()