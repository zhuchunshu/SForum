<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
namespace App\CodeFec\Menu;

use Illuminate\Support\Arr;

class Menu implements MenuInterface
{
    public $list = [
        
    ];

    public function get(): array
    {
        $array = $this->list;
        ksort($array);
        return $array;
    }

    public function add(int $id, array $arr): bool
    {
        $this->list = Arr::add($this->list, $id, $arr);
        return true;
    }
}
