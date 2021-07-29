<?php


namespace App\Plugins\Core\src\Lib;


class Redirect
{
    public string $url;

    public function url($url): Redirect
    {
      $this->url = $url;
      return $this;
    }

    public function back(): Redirect
    {
        $this->url = session()->get("_previous")['url'];
        return $this;
    }

    public function with($key, $value): Redirect
    {
        session()->flash($key, $value);
        return $this;
    }

    public function go(): \Psr\Http\Message\ResponseInterface
    {
        return response()->redirect($this->url);
    }
}