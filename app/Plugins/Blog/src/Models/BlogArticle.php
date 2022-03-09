<?php

declare (strict_types=1);
namespace App\Plugins\Blog\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $blog_id 
 * @property string $title 
 * @property string $class_id 
 * @property string $content 
 * @property string $markdown 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class BlogArticle extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'blog_article';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','blog_id','title','class_id','content','markdown','created_at', 'updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
	
	public function Blog(){
		return $this->belongsTo(Blog::class,'blog_id','id');
	}
}