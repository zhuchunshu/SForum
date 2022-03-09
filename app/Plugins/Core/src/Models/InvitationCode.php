<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $code 
 * @property string $user_id 
 * @property int $status 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class InvitationCode extends Model
{
	
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'invitation_code';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','code','user_id','status','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'status' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
	
	public function user(): \Hyperf\Database\Model\Relations\BelongsTo
	{
		return $this->belongsTo(User::class,"user_id","id");
	}
	
}