<?php

declare (strict_types=1);
namespace App\Plugins\Blog\src\Models;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $user_id 
 * @property string $token 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class Blog extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'blog';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','user_id','token','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
	
	protected $hidden = ['token'];
	
	public function user(){
		return $this->belongsTo(User::class,'user_id','id');
	}
}