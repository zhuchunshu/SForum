<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
Itf()->add('ui-topic-create-comment-before-hook', 1, [
    'enable' => (function () {
        return true;
    }),
    'view' => 'App::extends.widgets.next_comment',
]);
