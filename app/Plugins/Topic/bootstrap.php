<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
menu()->add(301, [
    'name' => '帖子标签',
    'url' => '#',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-tags" viewBox="0 0 16 16">
  <path d="M3 2v4.586l7 7L14.586 9l-7-7H3zM2 2a1 1 0 0 1 1-1h4.586a1 1 0 0 1 .707.293l7 7a1 1 0 0 1 0 1.414l-4.586 4.586a1 1 0 0 1-1.414 0l-7-7A1 1 0 0 1 2 6.586V2z"/>
  <path d="M5.5 5a.5.5 0 1 1 0-1 .5.5 0 0 1 0 1zm0 1a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3zM1 7.086a1 1 0 0 0 .293.707L8.75 15.25l-.043.043a1 1 0 0 1-1.414 0l-7-7A1 1 0 0 1 0 7.586V3a1 1 0 0 1 1-1v5.086z"/>
</svg>',
]);

// 菜单
menu()->add(302, [
    'name' => '管理',
    'url' => '/admin/topic/tag',
    'icon' => '',
    'parent_id' => 301,
]);

// 菜单
menu()->add(303, [
    'name' => '新增',
    'url' => '/admin/topic/tag/create',
    'icon' => '',
    'parent_id' => 301,
]);

menu()->add(304, [
    'name' => '任务',
    'url' => '/admin/topic/tag/jobs',
    'icon' => '',
    'parent_id' => 301,
]);

// 首页菜单
Itf()->add('menu', 1, [
    'name' => 'app.home',
    'url' => '/',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-home" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <polyline points="5 12 3 12 12 3 21 12 19 12"></polyline>
   <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"></path>
   <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"></path>
</svg>',
]);

Itf()->add('menu', 101, [
    'name' => 'app.tag',
    'url' => '/tags',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-tag" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M11 3l9 9a1.5 1.5 0 0 1 0 2l-6 6a1.5 1.5 0 0 1 -2 0l-9 -9v-4a4 4 0 0 1 4 -4h4"></path>
   <circle cx="9" cy="9" r="2"></circle>
</svg>',
]);

Itf()->add('menu', 419, [
    'name' => 'app.report center',
    'url' => '/report',
    'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-flag-3" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M5 14h14l-4.5 -4.5l4.5 -4.5h-14v16"></path>
</svg>',
    'quanxian' => 1000,
]);

Itf_Setting()->add(
    3001,
    '帖子设置',
    'topic',
    'Topic::admin.setting'
);

Itf_Setting()->add(
    23,
    '内容渲染设置',
    'sign',
    'Topic::admin.setting.contentParse'
);

// 权限
Authority()->add('topic_tag_create', '创建标签');

// topic create view

Itf()->add('topic-create-data', 0, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'Topic::create.basis',
    'scripts' => [
        file_hash('plugins/Topic/js/topic.js'),
        file_hash('plugins/Topic/js/editor.js'),
        file_hash('tabler/libs/tom-select/dist/js/tom-select.base.min.js'),
        file_hash('tabler/libs/tinymce/tinymce.min.js'),
    ],
]);

// topic create -  editor plugins

Itf()->add('topic-create-editor-plugins', 0, ['importcss', 'searchreplace', 'autolink', 'autosave',  'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'media', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'anchor', 'insertdatetime', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars',]);

Itf()->add('topic-create-editor-toolbar', 0, ['undo', 'redo', 'restoredraft', '|', 'bold', 'italic', 'underline', 'strikethrough', '|', 'fontfamily', 'fontsize', 'blocks', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify','outdent', 'indent','numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', '|', 'insertfile', 'image', 'media','link', 'anchor', 'codesample', '|', 'ltr', 'rtl']);

Itf()->add('topic-create-editor-toolbar', 1, [
    'basicDateButton',
]);

Itf()->add('topic-create-editor-menu', 0, [
    'file' => [
        'title' => 'File',
        'items' => [
            'newdocument',
            'restoredraft',
            '|',
            'preview',
            '|',
            'export',
            'print',
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
]);

Itf()->add('topic-create-editor-menu', 1, [
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
            '|',
            'insertdatetime',
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
]);

//Itf()->add('topic-create-options',0,[
//    'enable' => (function(){
//        return true;
//    }),
//    'view' => 'Topic::create.options.noupload',
//    'scripts' => [
//        //file_hash('tabler/libs/dropzone/dist/dropzone-min.js')
//    ]
//]);

Itf()->add('topic-create-handle-middleware-end',0,\App\Plugins\Topic\src\Handler\Topic\Middleware\Create\CreateEndMiddleware::class);