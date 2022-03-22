<?php

namespace App\Controller;

use App\CodeFec\Annotation\RouteRewrite;
use App\CodeFec\Annotation\ShortCode\ShortCodeR;
use Hyperf\Di\Annotation\AnnotationCollector;
use Hyperf\HttpServer\Annotation\AutoController;

#[AutoController(prefix: "/test")]
class TestController
{
	public function index(){
		$arr = Itf()->get("ShortCodeR");
		$shortCodeR = AnnotationCollector::getMethodsByAnnotation(\App\CodeFec\Annotation\ShortCode\ShortCodeR::class);
		foreach ($shortCodeR as $data){
			$name = $data['annotation']->name;
			$callback = $data['class']."@".$data['method'];
			$arr[$name]=['callback' => $callback];
		}
		return $arr;
	}
}