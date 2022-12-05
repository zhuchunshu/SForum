<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
use App\CodeFec\Header\functions;
use App\CodeFec\Ui\functions as UiFunctions;

functions::header()->add(1, 0, 'admin.component.header.server_logger');
// UiFunctions::Ui()->add(111,"js",mix("js/admin/component.js"));
