<?php


namespace App\CodeFec\Itf\Route;


use Illuminate\Support\Arr;

class Route implements RouteInterface
{

    public $list=[];

    public function set($route, $callback)
    {
        $this->list = Arr::add($this->list, $route, $callback);
    }
	
	public function re($route, $callback): bool
	{
		$this->list[$route] = $callback;
		return true;
	}

    public function get(): array
    {
        return $this->list;
    }
}