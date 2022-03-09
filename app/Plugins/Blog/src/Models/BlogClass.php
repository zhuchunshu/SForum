<?php

declare (strict_types=1);
namespace App\Plugins\Blog\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $blog_id 
 * @property string $name 
 * @property string $parent_id 
 * @property string $token 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class BlogClass extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'blog_class';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','blog_id','name','parent_id','token','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
	
	protected $hidden = ['token'];
	
	public function parent(): \Hyperf\Database\Model\Relations\BelongsTo
	{
		return $this->belongsTo(__CLASS__,"parent_id","id");
	}
}