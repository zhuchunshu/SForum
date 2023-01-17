<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Search;

/**
 * Class Search.
 * @name Search
 * @version 1.0.0
 * @see https://github.com/zhuchunshu/sf-search
 */
class Search
{
    public function handler()
    {
        Itf()->add('ui-common-header-right-hook', 1, [
            'enable' => (function () {
                return true;
            }),
            'view' => 'Search::search',
        ]);
    }
}
