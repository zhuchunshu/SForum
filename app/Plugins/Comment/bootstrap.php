<?php

Itf()->add('create-topic-comment-handle-middleware-end', 0, \App\Plugins\Comment\src\Handler\Middleware\Create\Topic\EndMiddleware::class);

Itf()->add('comment-topic-create-editor-external_plugins', 0, [
    'sfPreview' => file_hash('plugins/Topic/js/editor/plugins/sfPreview.js'),
]);
Itf()->add('comment-topic-edit-editor-external_plugins', 0, [
    'sfPreview' => file_hash('plugins/Topic/js/editor/plugins/sfPreview.js'),
]);
Itf()->add('comment-topic-create-editor-plugins', 0, ['importcss', 'searchreplace', 'autolink', 'autosave', 'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'media', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars']);
Itf()->add('comment-topic-edit-editor-plugins', 0, ['importcss', 'searchreplace', 'autolink', 'autosave', 'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'media', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars']);

Itf()->add('comment-topic-create-editor-toolbar', 0, ['undo', 'redo', '|', 'blocks', '|', 'bold', 'italic', 'underline', 'strikethrough', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify', 'outdent', 'indent', 'numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', 'insertfile', 'image', 'media', 'link', 'sfPreview', 'restoredraft', 'codesample', '|', 'ltr', 'rtl']);
Itf()->add('comment-topic-edit-editor-toolbar', 0, ['undo', 'redo', '|', 'blocks', '|', 'bold', 'italic', 'underline', 'strikethrough', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify', 'outdent', 'indent', 'numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', 'insertfile', 'image', 'media', 'link', 'sfPreview', 'restoredraft', 'codesample', '|', 'ltr', 'rtl']);


$editor_menu =[
    'file' => [
        'title' => 'File',
        'items' => [
            'newdocument',
            'restoredraft',
            '|',
            'sfPreview',
            'export',
            '|',
            'deleteallconversations',
        ],
    ],
    'edit' => [
        'title' => 'Edit',
        'items' => [
            'undo',
            'redo',
            '|',
            'cut',
            'copy',
            'paste',
            'pastetext',
            '|',
            'selectall',
            '|',
            'searchreplace',
        ],
    ],
    'view' => [
        'title' => 'View',
        'items' => [
            'code',
            '|',
            'visualaid',
            'visualchars',
            'visualblocks',
            '|',
            'spellchecker',
            '|',
            'preview',
            '|',
            'showcomments',
        ],
    ],
    'insert' => [
        'title' => 'Insert',
        'items' => [
            'image',
            'link',
            'media',
            'addcomment',
            'pageembed',
            'template',
            'codesample',
            'inserttable',
            '|',
            'charmap',
            'emoticons',
            'hr',
            '|',
            'pagebreak',
            'nonbreaking',
            'anchor',
            'tableofcontents',
        ],
    ],
    'format' => [
        'title' => 'Format',
        'items' => [
            'bold',
            'italic',
            'underline',
            'strikethrough',
            'superscript',
            'subscript',
            'codeformat',
            '|',
            'styles',
            'blocks',
            'fontfamily',
            'fontsize',
            'align',
            'lineheight',
            '|',
            'forecolor',
            'backcolor',
            '|',
            'language',
            '|',
            'removeformat',
        ],
    ],
    'tools' => [
        'title' => 'Tools',
        'items' => [
            'spellchecker',
            'spellcheckerlanguage',
            '|',
            'a11ycheck',
            'code',
            'wordcount',
        ],
    ],
    'table' => [
        'title' => 'Table',
        'items' => [
            'inserttable',
            '|',
            'cell',
            'row',
            'column',
            '|',
            'advtablesort',
            '|',
            'tableprops',
            'deletetable',
        ],
    ],
];
Itf()->add('comment-topic-create-editor-menu', 0, $editor_menu);
Itf()->add('comment-topic-edit-editor-menu', 0, $editor_menu);