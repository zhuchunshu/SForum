<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;

/**
 */
class UsersAuth extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'users_auth';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','user_id','token','online','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
	
	public function user(){
		return  $this->belongsTo(User::class,"user_id","id");
	}
}