<?php


namespace App\CodeFec\Itf\Route;


use Hyperf\Collection\Arr;

class Route implements RouteInterface
{

    public $list=[];

    public function set($route, $callback)
    {
        $this->list = Arr::set($this->list, $route, $callback);
    }
	
	public function re($route, $callback): bool
	{
		$this->list[$route] = $callback;
		return true;
	}
	
	public function del($route): bool
	{
		$this->list = array_diff_key($this->list, [$route => $this->list[$route]]);
		return true;
	}

    public function get(): array
    {
        return $this->list;
    }
}