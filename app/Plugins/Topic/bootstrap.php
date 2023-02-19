<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
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
    'quanxian' => (function () {
        return true;
    }),
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
        file_hash('tabler/libs/tom-select/dist/js/tom-select.base.min.js'),
        file_hash('tabler/libs/tinymce/tinymce.min.js'),
    ],
]);

// topic create -  editor plugins

Itf()->add('topic-create-editor-plugins', 0, ['emoticons','importcss', 'searchreplace', 'autolink', 'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars']);
Itf()->add('topic-edit-editor-plugins', 0, ['emoticons','importcss', 'searchreplace', 'autolink', 'directionality', 'code', 'visualblocks', 'visualchars', 'image', 'link', 'codesample', 'table', 'charmap', 'pagebreak', 'nonbreaking', 'advlist', 'lists', 'wordcount', 'charmap', 'quickbars']);

Itf()->add('topic-create-editor-toolbar', 0, ['undo', 'redo', '|', 'blocks', '|','emoticons', 'bold', 'italic', 'underline', 'strikethrough', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify', 'outdent', 'indent', 'numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', 'insertfile', 'image', 'link', 'sfPreview', 'codesample', '|', 'ltr', 'rtl']);
Itf()->add('topic-edit-editor-toolbar', 0, ['undo', 'redo', '|', 'blocks', '｜','emoticons', 'bold', 'italic', 'underline', 'strikethrough', '|', 'alignleft', 'aligncenter', 'alignright', 'alignjustify', 'outdent', 'indent', 'numlist', 'bullist', '|', 'forecolor', 'backcolor', 'removeformat', 'insertfile', 'image', 'link', 'sfPreview', 'codesample', '|', 'ltr', 'rtl']);

$editor_menu = [
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
            'emoticons',
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
Itf()->add('topic-create-editor-menu', 0, $editor_menu);
Itf()->add('topic-edit-editor-menu', 0, $editor_menu);

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
Itf()->add('topic-create-editor-menu', 2, $editor_menu2);
Itf()->add('topic-edit-editor-menu', 2, $editor_menu2);

// 外部插件
$external_plugins = [
    'sf-code' => file_hash('plugins/Topic/js/editor/plugins/shortCode/code.js'),
    'sf-hidden' => file_hash('plugins/Topic/js/editor/plugins/shortCode/hidden.js'),
    'sf-alert' => file_hash('plugins/Topic/js/editor/plugins/shortCode/alert.js'),
    'sf-quote' => file_hash('plugins/Topic/js/editor/plugins/shortCode/quote.js'),
];

Itf()->add('topic-create-editor-external_plugins', 2, $external_plugins);
Itf()->add('topic-edit-editor-external_plugins', 2, $external_plugins);



// 编辑器选项
Itf()->add('topic-create-options', 0, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'Topic::create.options.disable_comment',
]);

Itf()->add('topic-create-handle-middleware-end', 0, \App\Plugins\Topic\src\Handler\Topic\Middleware\Create\CreateEndMiddleware::class);
Itf()->add('topic-edit-handle-middleware-end', 0, \App\Plugins\Topic\src\Handler\Topic\Middleware\Update\UpdateEndMiddleware::class);

Itf()->add('topic-create-editor-external_plugins', 0, [
    'sfPreview' => file_hash('plugins/Topic/js/editor/plugins/sfPreview.js'),
]);
Itf()->add('topic-edit-editor-external_plugins', 0, [
    'sfPreview' => file_hash('plugins/Topic/js/editor/plugins/sfPreview.js'),
]);

Itf()->add('topic-edit-data', 0, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'Topic::edit.basis',
    'scripts' => [
        file_hash('plugins/Topic/js/topic.js'),
        file_hash('tabler/libs/tom-select/dist/js/tom-select.base.min.js'),
        file_hash('tabler/libs/tinymce/tinymce.min.js'),
    ],
]);

Itf()->add('topic-edit-options', 0, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'Topic::edit.options.disable_comment',
]);

Itf()->add('topic-create-options', 1, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'Topic::create.options.only_author',
]);
Itf()->add('topic-edit-options', 1, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'Topic::edit.options.only_author',
]);

Itf()->add('topic-create-handle-middleware-end', 1, \App\Plugins\Topic\src\Handler\Topic\Middleware\Create\Options\OnlyAuthor::class);
Itf()->add('topic-edit-handle-middleware-end', 1, \App\Plugins\Topic\src\Handler\Topic\Middleware\Create\Options\OnlyAuthor::class);




// 新增帖子操作按钮 - 修改
Itf()->add('ui-topic-show-dropdown', 1, [
    'enable' => (function ($data) {
        if (Authority()->check('admin_topic_edit') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            return true;
        }
        if (Authority()->check('topic_edit') && auth()->id() === $data->user->id) {
            return true;
        }
        return false;
    }),
    'view' => (function ($data) {
        $__app_revise = __('app.revise');
        return <<<HTML
<a class="dropdown-item" href="/topic/{$data->id}/edit"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-pencil-square" viewBox="0 0 16 16">
                        <path d="M15.502 1.94a.5.5 0 0 1 0 .706L14.459 3.69l-2-2L13.502.646a.5.5 0 0 1 .707 0l1.293 1.293zm-1.75 2.456-2-2L4.939 9.21a.5.5 0 0 0-.121.196l-.805 2.414a.25.25 0 0 0 .316.316l2.414-.805a.5.5 0 0 0 .196-.12l6.813-6.814z"/>
                        <path fill-rule="evenodd" d="M1 13.5A1.5 1.5 0 0 0 2.5 15h11a1.5 1.5 0 0 0 1.5-1.5v-6a.5.5 0 0 0-1 0v6a.5.5 0 0 1-.5.5h-11a.5.5 0 0 1-.5-.5v-11a.5.5 0 0 1 .5-.5H9a.5.5 0 0 0 0-1H2.5A1.5 1.5 0 0 0 1 2.5v11z"/>
                    </svg>{$__app_revise}</a>
HTML;
    }),
]);

// 新增帖子操作按钮 - 精华 和置顶
Itf()->add('ui-topic-show-dropdown', 3, [
    'enable' => (function ($data) {
        if (auth()->check() && Authority()->check("topic_options")) {
            return true;
        }
        return false;
    }),
    'view' => (function ($data) {
        $__app_top = __('app.top');
        $__app_essence = __('app.essence');
        return <<<HTML
<a class="dropdown-item" core-click="topic-topping" topic-id="{$data->id}" >
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-arrow-up-circle" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0"></path>
   <path d="M12 8l-4 4"></path>
   <path d="M12 8l0 8"></path>
   <path d="M16 12l-4 -4"></path>
</svg>

{$__app_top}
</a>
<a class="dropdown-item" core-click="topic-essence" topic-id="{$data->id}" >
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-infinity" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M9.828 9.172a4 4 0 1 0 0 5.656a10 10 0 0 0 2.172 -2.828a10 10 0 0 1 2.172 -2.828a4 4 0 1 1 0 5.656a10 10 0 0 1 -2.172 -2.828a10 10 0 0 0 -2.172 -2.828"></path>
</svg>

{$__app_essence}
</a>
HTML;
    }),
]);



// 新增 锁帖按钮
Itf()->add('ui-topic-show-dropdown', 4, [
    'enable' => (function ($data) {
        if (auth()->check() && Authority()->check("topic_lock")) {
            return true;
        }
        return false;
    }),
    'view' => (function ($data) {
        if($data->status!==['lock']){
            return <<<HTML
<a class="dropdown-item" core-click="topic-lock" topic-id="{$data->id}" >
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-lock" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M5 11m0 2a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z"></path>
   <path d="M12 16m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"></path>
   <path d="M8 11v-4a4 4 0 0 1 8 0v4"></path>
</svg>
锁帖
</a>
HTML;
        }else{
            return <<<HTML
<a class="dropdown-item" core-click="topic-lock" topic-id="{$data->id}" >
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-lock-open" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M5 11m0 2a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z"></path>
   <path d="M12 16m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"></path>
   <path d="M8 11v-5a4 4 0 0 1 8 0"></path>
</svg>
解锁
</a>
HTML;
        }
    }),
]);


// 新增帖子操作按钮 - 删除
Itf()->add('ui-topic-show-dropdown', 100, [
    'enable' => (function ($data) {
        if (Authority()->check('admin_topic_delete') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            return true;
        }
        if (Authority()->check('topic_delete') && auth()->id() === $data->user->id) {
            return true;
        }
        return false;
    }),
    'view' => (function ($data) {
        $__app_delete = __('app.delete');
        return <<<HTML
<a class="dropdown-item text-danger" core-click="topic-delete" topic-id="{$data->id}" >
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-trash" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <line x1="4" y1="7" x2="20" y2="7"></line>
                        <line x1="10" y1="11" x2="10" y2="17"></line>
                        <line x1="14" y1="11" x2="14" y2="17"></line>
                        <path d="M5 7l1 12a2 2 0 0 0 2 2h8a2 2 0 0 0 2 -2l1 -12"></path>
                        <path d="M9 7v-3a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v3"></path>
                    </svg>
{$__app_delete}</a>
HTML;
    }),
]);


// 新增帖子操作按钮 - 举报
Itf()->add('ui-topic-show-dropdown', 99, [
    'enable' => (function ($data) {
        return true;
    }),
    'view' => (function ($data) {
        $__app_delete = __('app.delete');
        return <<<HTML
<a class="dropdown-item" data-bs-toggle="modal" data-bs-target="#modal-report" core-click="report-topic" topic-id="$data->id" >
<svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-flag-3"
                         width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                         stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M5 14h14l-4.5 -4.5l4.5 -4.5h-14v16"></path>
                    </svg>
                    举报
                    </a>
HTML;
    }),
]);
