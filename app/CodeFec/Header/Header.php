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
namespace App\CodeFec\Header;

use Illuminate\Support\Arr;

class Header implements HeaderInterface
{
    public $list = [
        // 0 =>[
        //     "type" => 0,
        //     "view" => "shared.header.left"
        // ],
        // 1 => [
        //     "type" => 1,
        //     "view" => "shared.header.right"
        // ]
    ];

    public function get(): array
    {
        return $this->list;
    }

    public function add(int $id, int $type, string $view): bool
    {
        $arr = [
            'type' => $type,
            'view' => $view,
        ];
        $this->list = Arr::add($this->list, $id, $arr);
        return true;
    }
}
