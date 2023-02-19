<?php

Itf()->add('create-topic-comment-handle-middleware-end', 0, \App\Plugins\Comment\src\Handler\Middleware\Create\Topic\EndMiddleware::class);
Itf()->add('edit-topic-comment-handle-middleware-end', 0, \App\Plugins\Comment\src\Handler\Middleware\Edit\Topic\EndMiddleware::class);


Itf()->add('comment-topic-create-editor-external_plugins', 0, [
    'sfPreview' => file_hash('plugins/Comment/js/editor/plugins/sfPreview.js'),
]);
Itf()->add('comment-topic-edit-editor-external_plugins', 0, [
    'sfPreview' => file_hash('plugins/Comment/js/editor/plugins/sfPreview.js'),
]);


Itf()->add('comment-topic-create-editor-plugins', 0, ['emoticons','importcss', 'searchreplace', 'autolink', 'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars']);
Itf()->add('comment-topic-edit-editor-plugins', 0, ['emoticons','importcss', 'searchreplace', 'autolink', 'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars']);

Itf()->add('comment-topic-create-editor-toolbar', 0, ['undo', 'redo', '|', 'blocks', '|','emoticons', 'bold', 'italic', 'underline', 'strikethrough', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify', 'outdent', 'indent', 'numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', 'insertfile', 'image', 'link', 'sfPreview', 'codesample', '|', 'ltr', 'rtl']);
Itf()->add('comment-topic-edit-editor-toolbar', 0, ['undo', 'redo', '|', 'blocks', '|','emoticons', 'bold', 'italic', 'underline', 'strikethrough', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify', 'outdent', 'indent', 'numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', 'insertfile', 'image', 'link', 'sfPreview', 'codesample', '|', 'ltr', 'rtl']);


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

// 删除评论
Itf()->add('ui-comment-show-dropdown',999,[
    'enable' => (function($data,$value){
        return true;
    }),
    'view' => (function($data,$comment){
        $tr = __("app.delete");
       return <<<HTML
        <a class="dropdown-item text-danger" comment-click="comment-delete-topic" comment-id="$comment->id"><svg xmlns="http://www.w3.org/2000/svg"
                         class="hvr-icon icon icon-tabler icon-tabler-trash" width="24"
                         height="24" viewBox="0 0 24 24" stroke-width="2"
                         stroke="currentColor" fill="none" stroke-linecap="round"
                         stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <line x1="4" y1="7" x2="20" y2="7"></line>
                        <line x1="10" y1="11" x2="10" y2="17"></line>
                        <line x1="14" y1="11" x2="14" y2="17"></line>
                        <path d="M5 7l1 12a2 2 0 0 0 2 2h8a2 2 0 0 0 2 -2l1 -12"></path>
                        <path d="M9 7v-3a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v3"></path>
                    </svg>{$tr}</a>
HTML;

    }),
]);

// 修改评论
Itf()->add('ui-comment-show-dropdown',1,[
    'enable' => (function($data,$value){
        return true;
    }),
    'view' => (function($data,$value){
        return <<<HTML
        <a class="dropdown-item" data-bs-toggle="tooltip" data-bs-placement="bottom" title="修改" href="/comment/topic/{$value->id}/edit">
<svg xmlns="http://www.w3.org/2000/svg"
                         class="hvr-icon icon icon-tabler icon-tabler-edit" width="24"
                         height="24" viewBox="0 0 24 24" stroke-width="2"
                         stroke="currentColor" fill="none" stroke-linecap="round"
                         stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M9 7h-3a2 2 0 0 0 -2 2v9a2 2 0 0 0 2 2h9a2 2 0 0 0 2 -2v-3"></path>
                        <path d="M9 15h3l8.5 -8.5a1.5 1.5 0 0 0 -3 -3l-8.5 8.5v3"></path>
                        <line x1="16" y1="5" x2="19" y2="8"></line>
                    </svg>
修改</a>
HTML;

    }),
]);

// 举报评论
Itf()->add('ui-comment-show-dropdown',2,[
    'enable' => (function($data,$value){
        return true;
    }),
    'view' => (function($data,$value,$comment){
        return <<<HTML
        <a class="dropdown-item" data-bs-toggle="modal" data-bs-target="#modal-report"
                   url="/{$data->id}.html/{$value->id}?page={$comment->currentPage()}" topic-id="{$data->id}"
                   comment-click="report-comment" comment-id="{$value->id}">
<svg xmlns="http://www.w3.org/2000/svg"
                         class="hvr-icon icon icon-tabler icon-tabler-flag-3" width="24"
                         height="24" viewBox="0 0 24 24" stroke-width="2"
                         stroke="currentColor" fill="none" stroke-linecap="round"
                         stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M5 14h14l-4.5 -4.5l4.5 -4.5h-14v16"></path>
                    </svg>
                    举报
</a>
HTML;

    }),
]);



// 自定义
$editor_menu2 = [
    'shortCode' => [
        'title' => '短代码',
        'items' => [
            'sf-code',
            'sf-hidden',
            'sf-alert',
            'sf-quote',
        ],
    ],
];
// 自定义编辑器
Itf()->add('comment-topic-create-editor-menu', 2, $editor_menu2);
Itf()->add('comment-topic-edit-editor-menu', 2, $editor_menu2);

// 外部插件
$external_plugins = [
    'sf-code' => file_hash('plugins/Comment/js/editor/plugins/shortCode/code.js'),
    'sf-hidden' => file_hash('plugins/Comment/js/editor/plugins/shortCode/hidden.js'),
    'sf-alert' => file_hash('plugins/Comment/js/editor/plugins/shortCode/alert.js'),
    'sf-quote' => file_hash('plugins/Comment/js/editor/plugins/shortCode/quote.js'),
];

Itf()->add('comment-topic-edit-editor-external_plugins', 2, $external_plugins);
Itf()->add('comment-topic-create-editor-external_plugins', 2, $external_plugins);
