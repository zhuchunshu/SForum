<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
Itf()->add('ui-topic-comment-before-hook', 1, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'App::topic.show.include.lfpage',
]);

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
