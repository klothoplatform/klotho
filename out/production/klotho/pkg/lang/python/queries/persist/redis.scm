(expression_statement
	(assignment
    	left: (identifier) @redisVar
        right: (call
        	function: [
            	(identifier) @funcCall ;; intended for direct imports 'Redis'
                (attribute ;; intended for regular module calls 'redis.Redis'
                	object: (identifier) @module
                    attribute: (identifier) @funcCall
                )
                (attribute ;; intended for cluster calls 'redis.cluster.RedisCluster'
                	object: (attribute
                    	object: (identifier) @module
                        attribute: (identifier) @subModule
                    )
                    attribute: (identifier) @funcCall
                )
            ]
            arguments: (argument_list) @args
        )
    )
) @expr 
;; This query looks to match the following
;; client = redis.Redis(host='localhost', port=6379, db=0)
;; client = redis.cluster.RedisCluster(host='localhost', port=6379)
;; client = Redis(host='localhost', port=6379, db=0)