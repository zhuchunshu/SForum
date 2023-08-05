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
    public array $list = [
        
    ];

    public function get(): array
    {
        $array = $this->list;
        ksort($array);
        return $array;
    }

    public function add(int $id, array $arr): bool
    {
        $this->list = Arr::set($this->list, $id, $arr);
        return true;
    }
	
	public function re($id,$data): bool
	{
		$this->list[$id] = $data;
		return true;
	}
	
	public function del($id): bool
	{
		$this->list = array_diff_key($this->list, [$id => $this->list[$id]]);
		return true;
	}
}
